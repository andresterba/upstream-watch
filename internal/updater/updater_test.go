package updater

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeDatabase is an in-memory Database used to exercise Updater's
// orchestration logic without a real SQL backend. Its query semantics
// mirror the real implementation in updateDB.go: GetEntry looks up an
// exact (name, commit) match, and GetLatestUpdatedEntry returns the
// highest-id row with updated = true.
type fakeDatabase struct {
	mu      sync.Mutex
	entries []Entry
	nextID  int64
}

func (f *fakeDatabase) AddEntry(e Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, existing := range f.entries {
		if existing.ModuleName == e.ModuleName && existing.Commit == e.Commit {
			return fmt.Errorf("duplicate entry for %s@%s", e.ModuleName, e.Commit)
		}
	}

	f.nextID++
	e.ID = f.nextID
	f.entries = append(f.entries, e)
	return nil
}

func (f *fakeDatabase) GetEntry(e Entry) (Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, existing := range f.entries {
		if existing.ModuleName == e.ModuleName && existing.Commit == e.Commit {
			return existing, nil
		}
	}

	return Entry{}, sql.ErrNoRows
}

func (f *fakeDatabase) GetLatestUpdatedEntry(moduleName string) (Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var latest Entry
	found := false
	for _, existing := range f.entries {
		if existing.ModuleName == moduleName && existing.Updated && (!found || existing.ID > latest.ID) {
			latest = existing
			found = true
		}
	}
	if !found {
		return Entry{}, sql.ErrNoRows
	}

	return latest, nil
}

// newGitRepo creates a throwaway git repository in a temp directory so
// Updater's git plumbing (rev-parse, diff) can be exercised for real.
func newGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")

	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}

	return strings.TrimSpace(string(out))
}

// commitFile writes name/content into dir and commits it, returning the
// resulting commit hash.
func commitFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	runGit(t, dir, "add", name)
	runGit(t, dir, "commit", "-m", "commit "+name)

	return runGit(t, dir, "rev-parse", "HEAD")
}

// TestNewUpdater_ReturnsErrorInsteadOfPanicking guards against a regression
// of a bug where NewUpdater panicked on a lookup failure (e.g. a directory
// that isn't a git repository), which crashed the whole daemon instead of
// letting the caller skip just that one module.
func TestNewUpdater_ReturnsErrorInsteadOfPanicking(t *testing.T) {
	notAGitRepo := t.TempDir()
	db := &fakeDatabase{}

	u, err := NewUpdater(notAGitRepo, nil, nil, nil, db)
	if err == nil {
		t.Fatal("expected NewUpdater() to return an error for a non-git directory, got nil")
	}
	if u != nil {
		t.Fatalf("expected nil Updater on error, got %+v", u)
	}
}

func TestNewUpdater_NewModuleStartsUnupdated(t *testing.T) {
	repo := newGitRepo(t)
	head := commitFile(t, repo, "file.txt", "v1")
	db := &fakeDatabase{}

	u, err := NewUpdater(repo, nil, nil, nil, db)
	if err != nil {
		t.Fatalf("NewUpdater() error = %v", err)
	}

	if u.dbEntry.Commit != head || u.dbEntry.Updated {
		t.Fatalf("dbEntry = %+v, want Commit=%s Updated=false", u.dbEntry, head)
	}
}

func TestUpdater_isUpdateNecessary(t *testing.T) {
	t.Run("necessary when the module was never updated before", func(t *testing.T) {
		repo := newGitRepo(t)
		commitFile(t, repo, "file.txt", "v1")
		db := &fakeDatabase{}

		u, err := NewUpdater(repo, nil, nil, nil, db)
		if err != nil {
			t.Fatalf("NewUpdater() error = %v", err)
		}

		necessary, err := u.isUpdateNecessary()
		if err != nil {
			t.Fatalf("isUpdateNecessary() error = %v", err)
		}
		if !necessary {
			t.Fatal("expected update to be necessary when there is no prior baseline")
		}
	})

	t.Run("not necessary when HEAD matches the last applied baseline", func(t *testing.T) {
		repo := newGitRepo(t)
		head := commitFile(t, repo, "file.txt", "v1")
		db := &fakeDatabase{}
		if err := db.AddEntry(Entry{ModuleName: repo, Commit: head, Updated: true}); err != nil {
			t.Fatalf("AddEntry() error = %v", err)
		}

		u, err := NewUpdater(repo, nil, nil, nil, db)
		if err != nil {
			t.Fatalf("NewUpdater() error = %v", err)
		}

		necessary, err := u.isUpdateNecessary()
		if err != nil {
			t.Fatalf("isUpdateNecessary() error = %v", err)
		}
		if necessary {
			t.Fatal("expected no update necessary when HEAD equals the recorded baseline")
		}
	})

	t.Run("necessary when HEAD has moved past the baseline", func(t *testing.T) {
		repo := newGitRepo(t)
		baseline := commitFile(t, repo, "file.txt", "v1")
		db := &fakeDatabase{}
		if err := db.AddEntry(Entry{ModuleName: repo, Commit: baseline, Updated: true}); err != nil {
			t.Fatalf("AddEntry() error = %v", err)
		}
		commitFile(t, repo, "file.txt", "v2")

		u, err := NewUpdater(repo, nil, nil, nil, db)
		if err != nil {
			t.Fatalf("NewUpdater() error = %v", err)
		}

		necessary, err := u.isUpdateNecessary()
		if err != nil {
			t.Fatalf("isUpdateNecessary() error = %v", err)
		}
		if !necessary {
			t.Fatal("expected update to be necessary once HEAD diverges from the baseline")
		}
	})
}

func TestUpdater_Update(t *testing.T) {
	t.Run("runs hooks and persists success on first update", func(t *testing.T) {
		repo := newGitRepo(t)
		head := commitFile(t, repo, "file.txt", "v1")
		db := &fakeDatabase{}

		preMarker := filepath.Join(repo, "pre-ran")
		updateMarker := filepath.Join(repo, "update-ran")
		postMarker := filepath.Join(repo, "post-ran")

		u, err := NewUpdater(
			repo,
			[]string{"touch " + preMarker},
			[]string{"touch " + updateMarker},
			[]string{"touch " + postMarker},
			db,
		)
		if err != nil {
			t.Fatalf("NewUpdater() error = %v", err)
		}

		if err := u.Update(); err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		for _, marker := range []string{preMarker, updateMarker, postMarker} {
			if _, err := os.Stat(marker); err != nil {
				t.Fatalf("expected hook to have run and created %s: %v", marker, err)
			}
		}

		got, err := db.GetEntry(Entry{ModuleName: repo, Commit: head})
		if err != nil {
			t.Fatalf("GetEntry() error = %v", err)
		}
		if !got.Updated {
			t.Fatalf("expected persisted entry to be marked updated, got %+v", got)
		}
	})

	t.Run("returns an error and persists nothing when already up to date", func(t *testing.T) {
		repo := newGitRepo(t)
		head := commitFile(t, repo, "file.txt", "v1")
		db := &fakeDatabase{}
		if err := db.AddEntry(Entry{ModuleName: repo, Commit: head, Updated: true}); err != nil {
			t.Fatalf("AddEntry() error = %v", err)
		}

		u, err := NewUpdater(repo, nil, nil, nil, db)
		if err != nil {
			t.Fatalf("NewUpdater() error = %v", err)
		}

		if err := u.Update(); err == nil {
			t.Fatal("expected Update() to report that no update was necessary")
		}
		if len(db.entries) != 1 {
			t.Fatalf("expected no new entry to be persisted, got %d entries", len(db.entries))
		}
	})

	t.Run("pre-hook failure aborts before running the update command", func(t *testing.T) {
		repo := newGitRepo(t)
		commitFile(t, repo, "file.txt", "v1")
		db := &fakeDatabase{}

		updateMarker := filepath.Join(repo, "update-ran")
		u, err := NewUpdater(repo, []string{"false"}, []string{"touch " + updateMarker}, nil, db)
		if err != nil {
			t.Fatalf("NewUpdater() error = %v", err)
		}

		if err := u.Update(); err == nil {
			t.Fatal("expected Update() to fail when the pre-hook fails")
		}
		if _, statErr := os.Stat(updateMarker); !os.IsNotExist(statErr) {
			t.Fatal("update command should not have run after a pre-hook failure")
		}
		if len(db.entries) != 0 {
			t.Fatalf("expected nothing persisted after a failed update, got %d entries", len(db.entries))
		}
	})

	t.Run("post-hook failure prevents persisting the update", func(t *testing.T) {
		repo := newGitRepo(t)
		commitFile(t, repo, "file.txt", "v1")
		db := &fakeDatabase{}

		u, err := NewUpdater(repo, nil, []string{"true"}, []string{"false"}, db)
		if err != nil {
			t.Fatalf("NewUpdater() error = %v", err)
		}

		if err := u.Update(); err == nil {
			t.Fatal("expected Update() to fail when the post-hook fails")
		}
		if len(db.entries) != 0 {
			t.Fatalf("expected nothing persisted when the post-hook fails, got %d entries", len(db.entries))
		}
	})
}
