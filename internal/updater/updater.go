package updater

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Updater struct {
	moduleName     string
	preHooks       []string
	updateCommands []string
	postHooks      []string
	db             Database
	dbEntry        Entry
}

func NewUpdater(path string, preHooks []string, updateCommands []string, postHooks []string, db Database) *Updater {
	u := &Updater{
		moduleName:     path,
		preHooks:       preHooks,
		updateCommands: updateCommands,
		postHooks:      postHooks,
		db:             db,
	}

	entry, err := u.getEntryForModule()
	if err != nil {
		panic(err)
	}

	u.dbEntry = *entry

	return u
}

func (u *Updater) Update() error {
	updateNecessary, err := u.isUpdateNecessary()
	if err != nil {
		return err
	}

	if !updateNecessary {
		return fmt.Errorf("no update for %s necessary", u.moduleName)
	}

	log.Printf("starting update of module %s\n", u.moduleName)
	err = u.executePreHooks()
	if err != nil {
		return err
	}

	err = u.executeUpdate()
	if err != nil {
		return err
	}

	err = u.executePostHooks()
	if err != nil {
		return err
	}

	err = u.persistExecutedUpdateInDB()
	if err != nil {
		return err
	}

	return nil
}

// isUpdateNecessary reports whether the module changed since the last commit
// it was successfully updated for. It diffs against that recorded baseline
// commit rather than assuming a `git pull` only ever brings in a single new
// commit (HEAD~1), which would silently miss changes further back whenever
// multiple commits are pulled at once.
func (u *Updater) isUpdateNecessary() (bool, error) {
	baselineEntry, err := u.db.GetLatestUpdatedEntry(u.moduleName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// never successfully updated before, so there is nothing to
			// diff against; treat this as an initial update.
			return true, nil
		}

		return false, fmt.Errorf("failed to determine if update is necessary %w", err)
	}

	runCommand := exec.Command("git", "diff", "--quiet", baselineEntry.Commit, "HEAD", "--", u.moduleName)
	runCommand.Dir = u.moduleName
	output, err := runCommand.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return true, nil
		}

		return false, fmt.Errorf("failed to determine if update is necessary %s\n output: %s", err, output)
	}

	return false, nil
}

func (u *Updater) executePreHooks() error {
	for _, hookCommand := range u.preHooks {
		hookCommandAsString := strings.Fields(hookCommand)
		output, err := u.executeCommand(hookCommandAsString)
		if err != nil {
			return fmt.Errorf("failed to execute pre-hook command %s\nerror: %s\noutput: %s", hookCommandAsString, err, output)
		}
	}

	return nil
}

func (u *Updater) executeUpdate() error {
	for _, hookCommand := range u.updateCommands {
		commandAsString := strings.Fields(hookCommand)
		output, err := u.executeCommand(commandAsString)
		if err != nil {
			return fmt.Errorf("failed to execute update command %s\nerror: %s\noutput: %s", commandAsString, err, output)
		}
	}

	return nil
}

func (u *Updater) executePostHooks() error {
	for _, hookCommand := range u.postHooks {
		hookCommandAsString := strings.Fields(hookCommand)
		output, err := u.executeCommand(hookCommandAsString)
		if err != nil {
			return fmt.Errorf("failed to execute post-hook command %s\nerror: %s\n output: %s", hookCommandAsString, err, output)
		}
	}

	return nil
}

func (u *Updater) executeCommand(commandWithArgs []string) (string, error) {
	// TODO: Change to CommandContext()
	runCommand := exec.Command(commandWithArgs[0], commandWithArgs[1:]...)
	runCommand.Dir = u.moduleName
	output, err := runCommand.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return "", nil
}

func (u *Updater) getCurrentCommitHash() (string, error) {
	runCommand := exec.Command("git", "rev-parse", "HEAD")
	runCommand.Dir = u.moduleName
	output, err := runCommand.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return strings.TrimSpace(string(output)), nil
}

func (u *Updater) getEntryForModule() (*Entry, error) {
	commitHash, err := u.getCurrentCommitHash()
	if err != nil {
		return nil, err
	}

	e := Entry{
		ModuleName: u.moduleName,
		Commit:     commitHash,
		Updated:    false,
	}

	foundEntry, err := u.db.GetEntry(e)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return &e, nil
		}

		return nil, err
	}

	return &foundEntry, nil
}

func (u *Updater) persistExecutedUpdateInDB() error {
	u.dbEntry.Updated = true
	err := u.db.AddEntry(u.dbEntry)
	if err != nil {
		return err
	}

	return nil
}
