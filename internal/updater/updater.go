package updater

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Updater struct {
	moduleName string
	preHooks   []string
	postHooks  []string
	db         Database
	dbEntry    Entry
}

func NewUpdater(path string, preHooks []string, postHooks []string, db Database) *Updater {
	u := &Updater{
		moduleName: path,
		preHooks:   preHooks,
		postHooks:  postHooks,
		db:         db,
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
		return fmt.Errorf("no update for %s neccesarry", u.moduleName)
	}

	log.Printf("starting update of module %s\n", u.moduleName)
	err = u.executePreHooks()
	if err != nil {
		return err
	}

	err = u.executePostHooks()
	if err != nil {
		return err
	}

	u.persistExecutedUpdateInDB()
	if err != nil {
		return err
	}

	return nil
}

func (u *Updater) isUpdateNecessary() (bool, error) {
	directoryChanged := false
	runCommand := exec.Command("git", "diff", "--quiet", "HEAD", "HEAD~1", "--", u.moduleName)
	output, err := runCommand.CombinedOutput()
	if err != nil {
		if err.Error() == "exit status 1" {
			directoryChanged = true
		} else {
			return false, fmt.Errorf("failed to determine if update is necessary %s\n output: %s", err, output)
		}
	}

	return directoryChanged && !u.dbEntry.Updated, nil
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

	return string(output), nil
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
