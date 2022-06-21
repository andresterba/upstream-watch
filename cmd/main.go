package main

import (
	"log"
	"os/exec"
	"time"

	"github.com/andresterba/upstream-watch/internal/config"
	"github.com/andresterba/upstream-watch/internal/files"
	"github.com/andresterba/upstream-watch/internal/updater"
)

func pullUpstreamRepository() {
	runCommand := exec.Command("git", "pull")
	output, err := runCommand.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to pull upstream repository\n%s\n", output)
	}

	log.Print("Successfully pulled upstream repository")
}

func updateSubdirectories(loadedConfig *config.Config, db updater.Database) {
	pullUpstreamRepository()

	ds := files.NewDirectoryScanner(loadedConfig.IgnoreFolders)
	directories, err := ds.ListDirectories()
	if err != nil {
		log.Fatalf("failed to list directories\n%s\n", err)
	}

	for _, subdirectory := range directories {
		updateConfig, err := config.GetUpdateConfig(subdirectory + "/.update-hooks.yaml")
		if err != nil {
			log.Printf("Failed to update submodule %s: %+v", subdirectory, err)
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

	<-time.After(loadedConfig.RetryIntervall * time.Second)
}

func updateRootRepository(loadedConfig *config.Config, db updater.Database) {
	pullUpstreamRepository()

	subdirectory := "."
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

	<-time.After(loadedConfig.RetryIntervall * time.Second)
}

func main() {
	for {
		loadedConfig, err := config.GetConfig(".upstream-watch.yaml")
		if err != nil {
			log.Fatal(err)
		}

		updateDb := updater.NewDatabase()

		rootDirectoryeMode := loadedConfig.SingleDirectoryMode

		switch rootDirectoryeMode {
		case true:
			updateRootRepository(loadedConfig, updateDb)

		case false:
			updateSubdirectories(loadedConfig, updateDb)
		}
	}
}
