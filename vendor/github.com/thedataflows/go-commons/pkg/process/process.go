package process

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/thedataflows/go-commons/pkg/stringutil"
)

// CurrentProcessDirectory returns the absolute directory of the current process
func CurrentProcessDirectory() string {
	exePath, _ := CurrentProcessPathE()
	return filepath.Dir(exePath)
}

// CurrentProcessPath returns the absolute path of the current running process or empty string
func CurrentProcessPath() string {
	exePath, _ := CurrentProcessPathE()
	return exePath
}

// CurrentProcessPathE returns the absolute path of the current running process or error
func CurrentProcessPathE() (string, error) {
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

// SetEnvPath appends (if before is true) or prepends element to PATH for the current process
func SetEnvPath(element string, before bool) error {
	if element == "" {
		return nil
	}
	path := os.Getenv("PATH")
	delim := ":"
	if runtime.GOOS == "windows" {
		delim = ";"
	}
	env := stringutil.ConcatStrings(path, delim, element)
	if before {
		env = stringutil.ConcatStrings(element, delim, path)
	}
	err := os.Setenv("PATH", env)
	if err != nil {
		return err
	}
	return nil
}

// GetRawEnv lookup a raw environment variable or return fallback value
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
