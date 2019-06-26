package daemon

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

type SystemdService struct {
	ServiceProperties
}

// GetTemplate - gets service config template
func (svc *SystemdService) GetTemplate() string {
	return systemDConfig
}

// SetTemplate - sets service config template
func (svc *SystemdService) SetTemplate(tplStr string) error {
	systemDConfig = tplStr
	return nil
}

// Standard service path for systemD daemons
func (svc *SystemdService) servicePath() string {
	return "/etc/systemd/system/" + svc.name + ".service"
}

// Is a service installed
func (svc *SystemdService) isInstalled() bool {

	if _, err := os.Stat(svc.servicePath()); err == nil {
		return true
	}

	return false
}

// Check service is running
func (svc *SystemdService) checkRunning() (string, bool) {
	output, err := exec.Command("systemctl", "status", svc.name+".service").Output()
	if err == nil {
		if matched, err := regexp.MatchString("Active: active", string(output)); err == nil && matched {
			reg := regexp.MustCompile("Main PID: ([0-9]+)")
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
func (svc *SystemdService) Install(args ...string) (string, error) {
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

	templ, err := template.New("systemDConfig").Parse(systemDConfig)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Dependencies, Path, Args string
		}{
			svc.name,
			svc.description,
			strings.Join(svc.dependencies, " "),
			execPatch,
			strings.Join(args, " "),
		},
	); err != nil {
		return installAction + failed, err
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return installAction + failed, err
	}

	if err := exec.Command("systemctl", "enable", svc.name+".service").Run(); err != nil {
		return installAction + failed, err
	}

	return installAction + success, nil
}

// Remove the service
func (svc *SystemdService) Remove() (string, error) {
	removeAction := "Removing " + svc.description + ":"

	if ok, err := checkPrivileges(); !ok {
		return removeAction + failed, err
	}

	if !svc.isInstalled() {
		return removeAction + failed, ErrNotInstalled
	}

	if err := exec.Command("systemctl", "disable", svc.name+".service").Run(); err != nil {
		return removeAction + failed, err
	}

	if err := os.Remove(svc.servicePath()); err != nil {
		return removeAction + failed, err
	}

	return removeAction + success, nil
}

// Start the service
func (svc *SystemdService) Start() (string, error) {
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

	if err := exec.Command("systemctl", "start", svc.name+".service").Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (svc *SystemdService) Stop() (string, error) {
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

	if err := exec.Command("systemctl", "stop", svc.name+".service").Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (svc *SystemdService) Status() (string, error) {

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
func (svc *SystemdService) Run(e Executable) (string, error) {
	runAction := "Running " + svc.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

var systemDConfig = `[Unit]
Description={{.Description}}
Requires={{.Dependencies}}
After={{.Dependencies}}

[Service]
PIDFile=/var/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /var/run/{{.Name}}.pid
ExecStart={{.Path}} {{.Args}}
Restart=on-failure

[Install]
WantedBy=multi-user.target
`
