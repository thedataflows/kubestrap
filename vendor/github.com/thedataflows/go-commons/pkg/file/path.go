package file

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/thedataflows/go-commons/pkg/process"
)

// TrimExtension returns file path without extension
func TrimExtension(fileName string) string {
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}

// AppendExtension appends exe if OS is windows
func AppendExtension(fileName string) string {
	if runtime.GOOS == "windows" {
		return fileName + ".exe"
	}
	return fileName
}

// AppHome returns a directory path in the user home and creates it if needed
//
// if programPath is empty, current running process is used to extract program name
func AppHome(programPath string) (string, error) {
	var err error
	if programPath == "" {
		programPath, err = process.CurrentProcessPathE()
		if err != nil {
			return "", err
		}
	}
	programName := TrimExtension(filepath.Base(programPath))
	userHome, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	programHome := filepath.Join(userHome, programName)
	err = os.MkdirAll(programHome, 0700)
	if err != nil {
		return programHome, err
	}
	return programHome, nil
}

// WorkingDirectory returns the current working directory or empty on error
func WorkingDirectory() string {
	dir, _ := os.Getwd()
	return dir
}
