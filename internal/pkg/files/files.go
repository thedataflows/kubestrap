package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"dataflows.com/kubestrap/internal/pkg/logging"
)

const BUFFERSIZE = 1024

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

// CurrentProcessPath returns the absolute path of the current running process
func CurrentProcessPath() (string, error) {
	exePath, errOsExePath := os.Executable()
	if errOsExePath != nil {
		return "", errOsExePath
	}
	p, errAbs := filepath.Abs(exePath)
	if errAbs != nil {
		return "", errAbs
	}
	return p, nil
}

// AppHome returns a directory path in the user home and creates it if needed
//
// if programPath is empty, current running process is used to extract program name
func AppHome(programPath string) (string, error) {
	var err error
	if programPath == "" {
		programPath, err = CurrentProcessPath()
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

// IsAccessible check if a file or dir is accessible
func IsAccessible(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory returns true if path exists and is a directory
func IsDirectory(path string) bool {
	d, err := os.Stat(path)
	return err == nil && d.IsDir()
}

// IsFile returns true if path exists and is a file
func IsFile(path string) bool {
	d, err := os.Stat(path)
	return err == nil && !d.IsDir()
}

// CopyFile copies contents of a file using specified buffer
func CopyFile(src, dst string, BUFFERSIZE int64, overwrite bool) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	if !overwrite {
		_, err = os.Stat(dst)
		if err == nil {
			return fmt.Errorf("file %s already exists", dst)
		}
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	buf := make([]byte, BUFFERSIZE)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	return err
}

// GetKubeconfigPath returns path to kubernetes config file
func GetKubeconfigPath() string {
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		env := "HOME"
		if runtime.GOOS == "windows" {
			env = "USERPROFILE"
		}
		kubeConfig = filepath.Join(os.Getenv(env), "/.kube/config")
	}
	if !IsFile(kubeConfig) {
		logging.Logger.Warnf("Kubernetes config '%s' is not a valid file\n", kubeConfig)
	}
	return kubeConfig
}
