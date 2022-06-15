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
}

func NewUpdater(path string, preHooks []string, postHooks []string) *Updater {
	return &Updater{
		moduleName: path,
		preHooks:   preHooks,
		postHooks:  postHooks,
	}
}

func (u *Updater) Update() error {
	updateNecessary, err := u.isUpdateNecessary()
	if err != nil {
		return err
	}

	if !updateNecessary {
		return nil
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

	return nil
}

func (u *Updater) isUpdateNecessary() (bool, error) {
	runCommand := exec.Command("git", "diff", "--quiet", "HEAD", "HEAD~1", "--", u.moduleName)
	output, err := runCommand.CombinedOutput()
	if err != nil {
		if err.Error() == "exit status 1" {
			return true, nil
		}

		return false, fmt.Errorf("failed to determine if update is necessary %s\n output: %s\n", err, output)
	}

	return false, nil
}

func (u *Updater) executePreHooks() error {
	for _, hookCommand := range u.preHooks {
		hookCommandAsString := strings.Fields(hookCommand)
		output, err := u.executeCommand(hookCommandAsString)
		if err != nil {
			return fmt.Errorf("failed to execute pre-hook command %s\nerror: %s\noutput: %s\n", hookCommandAsString, err, output)
		}
	}

	return nil
}

func (u *Updater) executePostHooks() error {
	for _, hookCommand := range u.postHooks {
		hookCommandAsString := strings.Fields(hookCommand)
		output, err := u.executeCommand(hookCommandAsString)
		if err != nil {
			return fmt.Errorf("failed to execute post-hook command %s\nerror: %s\n output: %s\n", hookCommandAsString, err, output)
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
