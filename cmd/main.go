package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/andresterba/upstream-watch/internal/config"
	"github.com/andresterba/upstream-watch/internal/files"
)

func main() {
	for {
		loadedConfig, err := config.GetConfig("config.yaml")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%+v\n", loadedConfig)

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

		for _, subdirectory := range directories {
			fmt.Printf("Subdir: %s\n", subdirectory)
			updateConfig, error := config.GetUpdateConfig(subdirectory + "/config.yaml")
			if err != nil {
				log.Fatal(error)
			}
			fmt.Printf("%+v\n", updateConfig)

			preCommands := strings.Fields(updateConfig.PreUpdateCommand)
			runCommand := exec.Command(preCommands[0], preCommands[1:]...)
			output, err := runCommand.CombinedOutput()
			if err != nil {
				log.Fatalf("failed to pull\n%s\n", output)
			}

			fmt.Printf("PreCommandOutput: %s\n", output)

			postCommands := strings.Fields(updateConfig.PostUpdateCommand)
			runPostCommand := exec.Command(postCommands[0], postCommands[1:]...)
			output, err = runPostCommand.CombinedOutput()
			if err != nil {
				log.Fatalf("failed to pull\n%s\n", output)
			}

			fmt.Printf("PostCommandOutput: %s\n", output)

		}

		<-time.After(5 * time.Second)
	}
}
