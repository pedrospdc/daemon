// Package daemon darwin (mac os x) version
package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"text/template"
)

type DarwinService struct {
	ServiceProperties
}

// GetTemplate - gets service config template
func (svc *DarwinService) GetTemplate() string {
	return propertyList
}

// SetTemplate - sets service config template
func (svc *DarwinService) SetTemplate(tplStr string) error {
	propertyList = tplStr
	return nil
}

func newDaemon(name, description string, dependencies []string) (Daemon, error) {
	return &Service{name, description, dependencies}, nil
}

// Standard service path for system daemons
func (svc *DarwinService) servicePath() string {
	return "/Library/LaunchDaemons/" + svc.name + ".plist"
}

// Is a service installed
func (svc *DarwinService) isInstalled() bool {
	if _, err := os.Stat(svc.servicePath()); err == nil {
		return true
	}

	return false
}

// Get executable path
func execPath() (string, error) {
	return filepath.Abs(os.Args[0])
}

// Check service is running
func (svc *DarwinService) checkRunning() (string, bool) {
	output, err := exec.Command("launchctl", "list", svc.name).Output()
	if err == nil {
		if matched, err := regexp.MatchString(svc.name, string(output)); err == nil && matched {
			reg := regexp.MustCompile("PID\" = ([0-9]+);")
			data := reg.FindStringSubmatch(string(output))
			if len(data) > 1 {
				return "Service (pid  " + data[1] + ") is running...", true
			}
			return "Service is running...", true
		}
	}

	return "Service is stopped", false
}

// Install the service
func (svc *DarwinService) Install(args ...string) (string, error) {
	installAction := "Install " + svc.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return installAction + failed, err
	}

	srvPath := svc.servicePath()

	if svc.isInstalled() {
		return installAction + failed, ErrAlreadyInstalled
	}

	file, err := os.Create(srvPath)
	if err != nil {
		return installAction + failed, err
	}
	defer file.Close()

	execPatch, err := executablePath(&svc.ServiceProperties)
	if err != nil {
		return installAction + failed, err
	}

	templ, err := template.New("propertyList").Parse(propertyList)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Path string
			Args       []string
		}{svc.name, execPatch, args},
	); err != nil {
		return installAction + failed, err
	}

	return installAction + success, nil
}

// Remove the service
func (svc *DarwinService) Remove() (string, error) {
	removeAction := "Removing " + svc.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return removeAction + failed, err
	}

	if !svc.isInstalled() {
		return removeAction + failed, ErrNotInstalled
	}

	if err := os.Remove(svc.servicePath()); err != nil {
		return removeAction + failed, err
	}

	return removeAction + success, nil
}

// Start the service
func (svc *DarwinService) Start() (string, error) {
	startAction := "Starting " + svc.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return startAction + failed, err
	}

	if !svc.isInstalled() {
		return startAction + failed, ErrNotInstalled
	}

	if _, ok := svc.checkRunning(); ok {
		return startAction + failed, ErrAlreadyRunning
	}

	if err := exec.Command("launchctl", "load", svc.servicePath()).Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (svc *DarwinService) Stop() (string, error) {
	stopAction := "Stopping " + svc.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return stopAction + failed, err
	}

	if !svc.isInstalled() {
		return stopAction + failed, ErrNotInstalled
	}

	if _, ok := svc.checkRunning(); !ok {
		return stopAction + failed, ErrAlreadyStopped
	}

	if err := exec.Command("launchctl", "unload", svc.servicePath()).Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (svc *DarwinService) Status() (string, error) {
	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if !svc.isInstalled() {
		return "Status could not defined", ErrNotInstalled
	}

	statusAction, _ := svc.checkRunning()

	return statusAction, nil
}

// Run - Run service
func (svc *DarwinService) Run(e Executable) (string, error) {
	runAction := "Running " + svc.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

var propertyList = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>KeepAlive</key>
	<true/>
	<key>Label</key>
	<string>{{.Name}}</string>
	<key>ProgramArguments</key>
	<array>
	    <string>{{.Path}}</string>
		{{range .Args}}<string>{{.}}</string>
		{{end}}
	</array>
	<key>RunAtLoad</key>
	<true/>
    <key>WorkingDirectory</key>
    <string>/usr/local/var</string>
    <key>StandardErrorPath</key>
    <string>/usr/local/var/log/{{.Name}}.err</string>
    <key>StandardOutPath</key>
    <string>/usr/local/var/log/{{.Name}}.log</string>
</dict>
</plist>
`
