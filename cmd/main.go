package main

import (
	"log"
	"os/exec"
	"time"

	"github.com/andresterba/upstream-watch/internal/config"
	"github.com/andresterba/upstream-watch/internal/files"
	"github.com/andresterba/upstream-watch/internal/updater"
)

func main() {
	for {
		loadedConfig, err := config.GetConfig("config.yaml")
		if err != nil {
			log.Fatal(err)
		}

		runCommand := exec.Command("git", "pull")
		output, err := runCommand.CombinedOutput()
		if err != nil {
			log.Fatalf("failed to pull\n%s\n", output)
		}

		log.Print("Successfully pulled!")

		ds := files.NewDirectoryScanner(loadedConfig.IgnoreFolders)
		directories, err := ds.ListDirectories()
		if err != nil {
			log.Fatalf("failed to list directories\n%s\n", output)
		}

		db := updater.NewDatabase()

		for _, subdirectory := range directories {
			updateConfig, err := config.GetUpdateConfig(subdirectory + "/config.yaml")
			if err != nil {
				log.Printf("Failed to update submodule %s: %+v", subdirectory, err)
			}

			updater := updater.NewUpdater(
				subdirectory,
				updateConfig.PreUpdateCommands,
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
}
