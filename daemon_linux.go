// Package daemon linux version
package daemon

import (
	"os"
)

// Get the daemon properly
func newDaemon(name, description string, arguments []string, dependencies []string) (Daemon, error) {
	// newer subsystem must be checked first
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return &SystemdService{
			ServiceProperties{
				name: name,
				description: description,
				arguments: arguments,
				dependencies: dependencies,
			},
		}, nil
	}
	if _, err := os.Stat("/sbin/initctl"); err == nil {
		return &UpstartService{
			ServiceProperties{
				name: name,
				description: description,
				arguments: arguments,
				dependencies: dependencies,
			},
		}, nil
	}
	return &SystemvService{
		ServiceProperties{
			name: name,
			description: description,
			arguments: arguments,
			dependencies: dependencies,
		},
	}, nil
}

// Get executable path
func execPath() (string, error) {
	return os.Readlink("/proc/self/exe")
}
