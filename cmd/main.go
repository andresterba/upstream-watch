package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/andresterba/upstream-watch/internal/config"
	"github.com/andresterba/upstream-watch/internal/files"
	"github.com/andresterba/upstream-watch/internal/updater"
)

func pullUpstreamRepository(runPath string) {
	runCommand := exec.Command("git", "pull")
	runCommand.Dir = runPath
	output, err := runCommand.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to pull upstream repository\n%s\n", output)
	}

	log.Print("Successfully pulled upstream repository")
}

func updateSubdirectories(runPath string, loadedConfig *config.Config, db updater.Database) {
	pullUpstreamRepository(runPath)

	ds := files.NewDirectoryScanner(loadedConfig.IgnoreFolders)
	directories, err := ds.ListDirectories(runPath)
	if err != nil {
		log.Fatalf("failed to list directories\n%s\n", err)
	}

	for _, subdirectory := range directories {
		updateConfig, err := config.GetUpdateConfig(subdirectory + "/.update-hooks.yaml")
		if err != nil {
			log.Printf("Failed to update submodule %s: %+v", subdirectory, err)
			continue
		}

		updater := updater.NewUpdater(
			subdirectory,
			updateConfig.PreUpdateCommands,
			updateConfig.UpdateCommands,
			updateConfig.PostUpdateCommands,
			db,
		)
		err = updater.Update()
		if err != nil {
			log.Printf("Failed to update submodule %s: %+v", subdirectory, err)
			continue
		}

		log.Printf("Successfully updated submodule %s", subdirectory)
	}

	<-time.After(loadedConfig.RetryInterval * time.Second)
}

func updateRootRepository(runPath string, loadedConfig *config.Config, db updater.Database) {
	pullUpstreamRepository(runPath)

	subdirectory := path.Join(runPath, "/")
	updateConfig, err := config.GetUpdateConfig(subdirectory + "/.update-hooks.yaml")
	if err != nil {
		log.Printf("Failed to update root: %+v", err)
	}

	updater := updater.NewUpdater(
		subdirectory,
		updateConfig.PreUpdateCommands,
		updateConfig.UpdateCommands,
		updateConfig.PostUpdateCommands,
		db,
	)
	err = updater.Update()
	if err != nil {
		log.Printf("Failed to update root: %+v", err)
	} else {
		log.Printf("Successfully updated root")
	}

	<-time.After(loadedConfig.RetryInterval * time.Second)
}

const configName = ".upstream-watch.yaml"

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Printf("Please provide specify the directory to run in as arg.\n")
		os.Exit(0)
	}

	runPath := os.Args[1]

	pathToConfig := path.Join(runPath, configName)

	for {
		loadedConfig, err := config.GetConfig(pathToConfig)
		if err != nil {
			log.Fatal(err)
		}

		updateDb := updater.NewDatabase()

		rootDirectoryeMode := loadedConfig.SingleDirectoryMode

		switch rootDirectoryeMode {
		case true:
			updateRootRepository(runPath, loadedConfig, updateDb)

		case false:
			updateSubdirectories(runPath, loadedConfig, updateDb)
		}
	}
}
