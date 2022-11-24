package kubestrap

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"dataflows.com/kubestrap/internal/pkg/files"
	"dataflows.com/kubestrap/internal/pkg/installer"
	"dataflows.com/kubestrap/internal/pkg/logging"
	"github.com/go-cmd/cmd"
	"golang.org/x/exp/slices"
)

type RawCommand struct {
	Name           string   `yaml:"name"`
	Additional     []string `yaml:"additional"`
	Command        []string `yaml:"command"`
	VersionCommand string   `yaml:"version-command"`
	Release        string   `yaml:"release"`
	Url            struct {
		Windows string `yaml:"windows"`
		Linux   string `yaml:"linux"`
		Darwin  string `yaml:"darwin"`
	} `yaml:"url"`
	Help      string `yaml:"help,omitempty"`
	CachePath string `yaml:"cache-path,omitempty"`
	Extract   struct {
		Pattern string   `yaml:"pattern"`
		List    []string `yaml:"list"`
	} `yaml:"extract,omitempty"`
}

// ExecuteCommand attempts to execute an instance of a subcommand
func (command *RawCommand) ExecuteCommand(timeout time.Duration, rawOutput bool, buffered bool) (int, error) {
	// Call this to set PATH
	_, errExeDir := command.ExeDir()
	if errExeDir != nil {
		return -1, logging.ErrWithTrace(errExeDir)
	}

	logging.ExitOnError(command.CheckCommand(timeout), -1)

	exePath, errLookup := exec.LookPath(command.Command[0])
	if errLookup != nil {
		return -2, logging.ErrWithTrace(errLookup)
	}

	status, errRun := RunProcess(exePath, command.Command[1:], timeout, rawOutput, buffered)
	if errRun != nil {
		return -3, logging.ErrWithTrace(errRun)
	}
	if buffered {
		for _, line := range status.Stdout {
			if rawOutput {
				fmt.Fprintln(os.Stdout, line)
			} else {
				logging.Logger.Infof("[%s] %v\n", command.Name, line)
			}
		}
		for _, line := range status.Stderr {
			if rawOutput {
				fmt.Fprintln(os.Stderr, line)
			} else {
				logging.Logger.Errorf("[%s] %v\n", command.Name, line)
			}
		}
	}
	return status.Exit, nil
}

// CheckCommand checks if the command exists in the PATH first, and if is at the specified version. Will attempt to download or get from filesystem and extract
func (command *RawCommand) CheckCommand(timeout time.Duration) error {
	var (
		output string
		status *cmd.Status
		errRun error
	)

	commandExePath, errLookup := exec.LookPath(command.Name)
	if errLookup != nil {
		goto getExe
	}

	// check version
	if command.VersionCommand == "" {
		command.VersionCommand = "version"
	}
	status, errRun = RunProcess(commandExePath, regexp.MustCompile(`\s+`).Split(command.VersionCommand, -1), timeout, true, true)
	if errRun != nil {
		return logging.ErrWithTrace(fmt.Errorf("[%s] version check failed:\n%+v", command.Name, errRun))
	}
	if status.Error != nil {
		return logging.ErrWithTrace(fmt.Errorf("[%s] version check failed:\n%+v", command.Name, status.Error))
	}
	if status.Exit != 0 {
		return logging.ErrWithTrace(fmt.Errorf("[%s] version check failed:\n%s", command.Name, strings.Join(status.Stderr, "\n")))
	}
	// some programs output version on stderr
	output = strings.Join(append(status.Stdout, status.Stderr...), "")
	if !strings.Contains(output, command.Release) {
		logging.Logger.Warnf("release '%s' was not matched in version command output:\n%s", command.Release, output)
		goto getExe
	}

	for i, p := range command.Additional {
		_, errLookup := exec.LookPath(p)
		if errLookup != nil {
			break
		}
		if i == len(command.Additional)-1 {
			return nil
		}
	}

getExe:
	exeList, errEnsureExe := command.EnsureExe()
	if errEnsureExe != nil {
		return errEnsureExe
	}
	exePath := ""
	for _, b := range exeList {
		if strings.HasSuffix(b, files.AppendExtension(command.Name)) {
			exePath = b
			break
		}
	}
	if exePath == "" {
		return fmt.Errorf("'%s' not found", files.AppendExtension(command.Name))
	}

	return nil
}

// EnsureExe will download and extract (if needed) specified or default version of an executable
//
// Returns list of files
func (command *RawCommand) EnsureExe() ([]string, error) {
	// check for error later, if we get to download
	parsedUrl, errParseUrl := command.GetUrl()
	if errParseUrl != nil {
		return nil, logging.ErrWithTrace(errParseUrl)
	}

	if parsedUrl.Path == "" && command.CachePath == "" {
		return nil, logging.ErrWithTrace(fmt.Errorf("at least one of 'url.%s' or 'cache-path' must be specified", runtime.GOOS))
	}

	download := false

	exeDir, errExeDir := command.ExeDir()
	if errExeDir != nil {
		return nil, logging.ErrWithTrace(errExeDir)
	}
	// Use cache first
	cachePath := command.CachePath
cache:
	if !filepath.IsAbs(cachePath) {
		cachePath = filepath.Join(exeDir, cachePath)
	}
	cachePathStat, errStat := os.Stat(cachePath)
	// cache invalid
	if errStat != nil {
		logging.Logger.Warnf("%+v", errStat)
		if parsedUrl.Path == "" {
			return nil, logging.ErrWithTrace(fmt.Errorf("'cache-path=%s' is invalid and 'url.%s' is empty", cachePath, runtime.GOOS))
		}
		download = true
	}
	// cache valid and is a directory
	if cachePathStat != nil && cachePathStat.IsDir() {
		commands := append([]string{command.Name}, command.Additional...)
		for i, c := range commands {
			if !files.IsAccessible(filepath.Join(cachePath, files.AppendExtension(c))) {
				// If one of the commands is not found, should download again
				break
			}
			if i == len(commands)-1 {
				return commands, nil
			}
		}
		if parsedUrl.Path == "" {
			return nil, logging.ErrWithTrace(fmt.Errorf("'cache-path=%s' is a directory but 'url.%s' is empty", cachePath, runtime.GOOS))
		}
		newCachePath := filepath.Join(cachePath, filepath.Base(parsedUrl.Path))
		if files.IsAccessible(newCachePath) {
			cachePath = newCachePath
			goto cache
		}
		download = true
	}
	// cache valid and is a file
	if cachePathStat != nil && !cachePathStat.IsDir() {
		// speed up by checking for no extension or .exe on Windows
		extensions := []string{""}
		if runtime.GOOS == "windows" {
			extensions = []string{".exe"}
		}
		if slices.Contains(extensions, filepath.Ext(cachePath)) {
			exePath := filepath.Join(exeDir, files.AppendExtension(command.Name))
			if cachePath == exePath {
				return []string{exePath}, nil
			}
			var err error
			if command.CachePath != "" || parsedUrl.Scheme == "file" {
				logging.Logger.Debugf("copying '%s' to '%s'", cachePath, exePath)
				err = files.CopyFile(cachePath, exePath, files.BUFFERSIZE, true)
			} else {
				logging.Logger.Debugf("moving '%s' to '%s'", cachePath, exePath)
				err = os.Rename(cachePath, exePath)
			}
			if err != nil {
				return nil, logging.ErrWithTrace(err)
			}
			return []string{exePath}, nil
		}
		// maybe is an archive?
		goto extract
	}

	if download {
		switch parsedUrl.Scheme {
		case "file":
			var err error
			cachePath, err = filepath.Abs(parsedUrl.Path)
			if err != nil {
				return nil, logging.ErrWithTrace(err)
			}
			_, err = os.Stat(cachePath)
			if err != nil {
				return nil, logging.ErrWithTrace(err)
			}
			goto cache
		case "http", "https":
			logging.Logger.Infof("downloading '%s' release '%s'", command.Name, command.Release)
			var errDownload error
			cachePath, errDownload = installer.DownloadFile(cachePath, parsedUrl.String())
			if errDownload != nil {
				return nil, logging.ErrWithTrace(errDownload)
			}
			goto cache
		default:
			return nil, logging.ErrWithTrace(fmt.Errorf("scheme '%s' not yet supported in '%s'. Please use 'file', 'http' or 'https'", parsedUrl.Scheme, parsedUrl.String()))
		}
	}

extract:
	listToExtract := command.Extract.List
	if len(command.Extract.List) == 0 {
		listToExtract = []string{files.AppendExtension(command.Name)}
	}
	extractedFiles, errExtract := installer.ExtractFiles(
		cachePath,
		exeDir,
		listToExtract,
		command.Extract.Pattern,
		true)
	if errExtract != nil {
		return nil, logging.ErrWithTrace(errExtract)
	}

	if download {
		err := os.Remove(cachePath)
		if err != nil {
			return nil, logging.ErrWithTrace(err)
		}
	}
	return extractedFiles, nil
}

// ExeDir returns cleaned and created command directory
func (command *RawCommand) ExeDir() (string, error) {
	appHome, errHome := files.AppHome("")
	if errHome != nil {
		return "", logging.ErrWithTrace(errHome)
	}
	dir := filepath.Clean(fmt.Sprintf("%s/bin/%s/%s", appHome, command.Name, command.Release))
	if !files.IsDirectory(dir) {
		errMkdir := os.MkdirAll(dir, 0700)
		if errMkdir != nil {
			return "", logging.ErrWithTrace(errMkdir)
		}
	}
	errEnv := SetEnvPath(dir, true)
	if errEnv != nil {
		return "", logging.ErrWithTrace(errEnv)
	}
	return dir, nil
}

// GetUrl returns platform specific url
func (command *RawCommand) GetUrl() (*url.URL, error) {
	var retUrl string
	switch runtime.GOOS {
	case "windows":
		retUrl = command.Url.Windows
	case "linux":
		retUrl = command.Url.Linux
	case "darwin":
		retUrl = command.Url.Darwin
	default:
		return nil, fmt.Errorf("unsupported platform '%s'", runtime.GOOS)
	}
	if retUrl != "" {
		retUrl = strings.ReplaceAll(retUrl, "{{release}}", command.Release)
		retUrl = strings.ReplaceAll(retUrl, "{{os}}", runtime.GOOS)
		retUrl = strings.ReplaceAll(retUrl, "{{arch}}", runtime.GOARCH)
		retUrl = strings.ReplaceAll(retUrl, "{{name}}", command.Name)
	}
	parsedUrl, errParseUrl := url.Parse(retUrl)
	if errParseUrl != nil {
		return nil, errParseUrl
	}
	return parsedUrl, nil
}
