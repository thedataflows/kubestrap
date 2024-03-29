package kubestrap

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/thedataflows/go-commons/pkg/log"
	"github.com/thedataflows/go-commons/pkg/stringutil"
	"github.com/thedataflows/kubestrap/pkg/constants"

	"github.com/go-cmd/cmd"
	"github.com/shirou/gopsutil/v3/process"
)

// RunProcess starts a process and waits for it to complete but not after specified timeout
func RunProcess(exePath string, args []string, timeout time.Duration, buffered bool, stdin io.Reader) (*cmd.Status, error) {
	// eliminate empty args
	var cleanArgs []string
	for _, a := range args {
		if a != "" {
			cleanArgs = append(cleanArgs, a)
		}
	}

	currentCmd := cmd.NewCmdOptions(cmd.Options{
		Buffered:       buffered,
		Streaming:      !buffered,
		LineBufferSize: 2 * cmd.DEFAULT_LINE_BUFFER_SIZE,
	}, exePath, cleanArgs...)

	exeName := filepath.Base(exePath)
	commandLineString := strings.Join(currentCmd.Args, " ")
	log.Debugf("command: %s %s", currentCmd.Name, commandLineString)

	// Check if process is already running
	pid, errIsProcessRunning := IsProcessRunning(currentCmd.Name, commandLineString)
	if errIsProcessRunning != nil {
		return nil, errIsProcessRunning
	}
	if pid > 0 {
		return nil, fmt.Errorf("'%s %s' is already running with PID '%v'", currentCmd.Name, commandLineString, pid)
	}

	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		// Done when both channels have been closed
		// https://dave.cheney.net/2013/04/30/curious-channels
		for currentCmd.Stdout != nil || currentCmd.Stderr != nil {
			select {
			case line, open := <-currentCmd.Stdout:
				if !open {
					currentCmd.Stdout = nil
					continue
				}
				if !buffered {
					log.Infof("[%s] %v", exeName, line)
				}
			case line, open := <-currentCmd.Stderr:
				if !open {
					currentCmd.Stderr = nil
					continue
				}
				if !buffered {
					log.Warnf("[%s] %v", exeName, line)
				}
			}
		}
	}()

	// Stop command after specified timeout
	if timeout > 0 {
		go func() {
			<-time.After(timeout)
			if !currentCmd.Status().Complete {
				err := currentCmd.Stop()
				if err != nil {
					log.Errorf("[%s] timeout running command after %v. Error: %v", exeName, timeout, err)
				} else {
					log.Errorf("[%s] timeout running command after %v", exeName, timeout)
				}
			}
		}()
	}

	// Run and wait for Cmd to return
	if stdin == nil {
		stdin = os.Stdin
	}
	statusChan := <-currentCmd.StartWithStdin(stdin)
	<-doneChan
	return &statusChan, nil
}

// IsProcessRunning returns the PID of a running process matched by image name and command line
func IsProcessRunning(binaryPath, cmdLine string) (int, error) {
	procs, err := process.Processes()
	if err != nil {
		return 0, err
	}
	command := filepath.Clean(binaryPath)
	if cmdLine != "" {
		command += " " + cmdLine
	}
	for _, p := range procs {
		processCmd, _ := p.Cmdline()
		if strings.Contains(processCmd, command) {
			return int(p.Pid), nil
		}
	}
	return 0, nil
}

// SetEnvPath appends (if before is true) or prepends element to PATH for the current process
func SetEnvPath(element string, before bool) error {
	if element == "" {
		return nil
	}
	path := os.Getenv("PATH")
	delim := ":"
	if runtime.GOOS == constants.Windows {
		delim = ";"
	}
	env := stringutil.ConcatStrings(path, delim, element)
	if before {
		env = stringutil.ConcatStrings(element, delim, path)
	}
	if err := os.Setenv("PATH", env); err != nil {
		return err
	}
	return nil
}
