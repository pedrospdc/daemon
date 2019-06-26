//+build go1.8

package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Service constants
const (
	success = "\t\t\t\t\t[  \033[32mOK\033[0m  ]" // Show colored "OK"
	failed  = "\t\t\t\t\t[\033[31mFAILED\033[0m]" // Show colored "FAILED"
)

var (
	// ErrUnsupportedSystem appears if try to use service on system which is not supported by this release
	ErrUnsupportedSystem = errors.New("unsupported system")

	// ErrRootPrivileges appears if run installation or deleting the service without root privileges
	ErrRootPrivileges = errors.New("you must have root user privileges. Possibly using 'sudo' command should help")

	// ErrAlreadyInstalled appears if service already installed on the system
	ErrAlreadyInstalled = errors.New("service has already been installed")

	// ErrNotInstalled appears if try to delete service which was not been installed
	ErrNotInstalled = errors.New("service is not installed")

	// ErrAlreadyRunning appears if try to start already running service
	ErrAlreadyRunning = errors.New("service is already running")

	// ErrAlreadyStopped appears if try to stop already stopped service
	ErrAlreadyStopped = errors.New("service has already been stopped")
)

// Lookup path for executable file
func executablePath(properties *ServiceProperties) (string, error) {
	var err error
	var foundPath string
	var path string

	if path, err = exec.LookPath(properties.name); err == nil {
		if _, err = os.Stat(path); err == nil {
			foundPath = path
		}
	}

	if foundPath == "" {
		foundPath, err = os.Executable()
	}

	if err != nil {
		return "", err
	}

	if foundPath != "" && len(properties.arguments) > 0 {
		return fmt.Sprintf("%s %s", foundPath, strings.Join(properties.arguments, " ")), nil
	}

	return "", nil
}

// Check root rights to use system service
func checkPrivileges() (bool, error) {
	if output, err := exec.Command("id", "-g").Output(); err == nil {
		if gid, parseErr := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32); parseErr == nil {
			if gid == 0 {
				return true, nil
			}
			return false, ErrRootPrivileges
		}
	}
	return false, ErrUnsupportedSystem
}
