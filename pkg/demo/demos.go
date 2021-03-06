package demo

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// File holds meta information about one demo file.
type File struct {
	MatchID   uint64
	MatchTime time.Time
	Filename  string
}

// ScanDemosDir scans the demos dir and returns all match ids.
func ScanDemosDir(path string) []*File {
	var demos []*File

	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Ignore non .dem-files
			if filepath.Ext(path) != ".dem" {
				return nil
			}

			fileName := info.Name()
			demoName := strings.TrimSuffix(fileName, filepath.Ext(fileName))

			matchID := getIDFromFileName(demoName)

			if matchID != 0 {
				// Get file creation date.
				modTime := time.Now()

				stats, err := os.Stat(path)
				if err != nil {
					log.Errorf("Unable to read file stats for %v", fileName)
				} else {
					modTime = stats.ModTime()
				}

				// Add demo
				demos = append(demos, &File{MatchID: matchID, MatchTime: modTime, Filename: fileName})
			}

			return nil
		})
	if err != nil {
		panic(err)
	}

	return demos
}

// Filename without extension.
func getIDFromFileName(demoName string) uint64 {
	// MatchID is not the MatchID used to request the game. It is similiar.
	// Demos downloaded by this tool are ${matchid}.dem
	// Demos downloaded by the game are match730_${matchid}_${outcomeid}_${token}.dem
	// Downloaded demos via a share code through the game are ${matchid}_${outcomeid}.dem

	var matchIDString string

	demoNameParts := strings.Split(demoName, "_")

	// Check for match730_${matchid}_${outcomeid}_${token}.dem
	reGameOwn := regexp.MustCompile(`^match730(_?[\d]{21})(_?[\d]{10})(_?[\d]{3})$`)
	foundGameOwn := reGameOwn.MatchString(demoName)

	if foundGameOwn {
		matchIDString = demoNameParts[1]
	}

	// Check for ${matchid}_${outcomeid}.dem
	reGameShareCode := regexp.MustCompile(`^(_?[\d]{21})(_?[\d]{10})$`)
	foundGameShareCode := reGameShareCode.MatchString(demoName)

	// Check for ${matchid}.dem
	reTool := regexp.MustCompile(`^[0-9]*$`)
	foundTool := reTool.MatchString(demoName)

	if foundGameShareCode || foundTool {
		matchIDString = demoNameParts[0]
	}

	if matchIDString == "" {
		return 0
	}

	matchID, err := strconv.ParseUint(matchIDString, 10, 64)

	if err != nil {
		return 0
	}

	return matchID
}
