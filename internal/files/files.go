package files

import (
	"fmt"
	"os"
)

type DirectoryScanner struct {
	directoriesToIngore map[string]struct{}
}

func NewDirectoryScanner(directoriesToIgnore []string) *DirectoryScanner {
	directoriesToIgnoreLookupMap := make(map[string]struct{})
	for _, dir := range directoriesToIgnore {
		_, ok := directoriesToIgnoreLookupMap[dir]
		if !ok {
			directoriesToIgnoreLookupMap[dir] = struct{}{}
		}
	}

	return &DirectoryScanner{
		directoriesToIngore: directoriesToIgnoreLookupMap,
	}
}

func (ds *DirectoryScanner) ListDirectories() ([]string, error) {
	subdirectories := []string{}
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to list directories %v", err)
	}

	for _, entry := range entries {
		_, ok := ds.directoriesToIngore[entry.Name()]
		if !ok {
			if entry.IsDir() {
				subdirectories = append(subdirectories, entry.Name())
			}
		}
	}

	return subdirectories, nil
}
