package daemon

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

// UpstartService - standard record (struct) for linux upstart version of daemon package
type UpstartService struct {
	ServiceProperties
}

// Run - Run service
func (svc *UpstartService) Run(e Executable) (string, error) {
	runAction := "Running " + svc.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

// GetTemplate - gets service config template
func (svc *UpstartService) GetTemplate() string {
	return upstatConfig
}

// SetTemplate - sets service config template
func (svc *UpstartService) SetTemplate(tplStr string) error {
	upstatConfig = tplStr
	return nil
}

// Standard service path for systemV daemons
func (svc *UpstartService) servicePath() string {
	return "/etc/init/" + svc.name + ".conf"
}

// Is a service installed
func (svc *UpstartService) isInstalled() bool {

	if _, err := os.Stat(svc.servicePath()); err == nil {
		return true
	}

	return false
}

// Check service is running
func (svc *UpstartService) checkRunning() (string, bool) {
	output, err := exec.Command("status", svc.name).Output()
	if err == nil {
		if matched, err := regexp.MatchString(svc.name+" start/running", string(output)); err == nil && matched {
			reg := regexp.MustCompile("process ([0-9]+)")
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
func (svc *UpstartService) Install(args ...string) (string, error) {
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

	templ, err := template.New("upstatConfig").Parse(upstatConfig)
	if err != nil {
		return installAction + failed, err
	}

	if err := templ.Execute(
		file,
		&struct {
			Name, Description, Path, Args string
		}{svc.name, svc.description, execPatch, strings.Join(args, " ")},
	); err != nil {
		return installAction + failed, err
	}

	if err := os.Chmod(srvPath, 0755); err != nil {
		return installAction + failed, err
	}

	return installAction + success, nil
}

// Remove the service
func (svc *UpstartService) Remove() (string, error) {
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
func (svc *UpstartService) Start() (string, error) {
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

	if err := exec.Command("start", svc.name).Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (svc *UpstartService) Stop() (string, error) {
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

	if err := exec.Command("stop", svc.name).Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (svc *UpstartService) Status() (string, error) {

	if ok, err := checkPrivileges(); !ok {
		return "", err
	}

	if !svc.isInstalled() {
		return "Status could not defined", ErrNotInstalled
	}

	statusAction, _ := svc.checkRunning()

	return statusAction, nil
}

var upstatConfig = `# {{.Name}} {{.Description}}

description     "{{.Description}}"
author          "Pichu Chen <pichu@tih.tw>"

start on runlevel [2345]
stop on runlevel [016]

respawn
#kill timeout 5

exec {{.Path}} {{.Args}} >> /var/log/{{.Name}}.log 2>> /var/log/{{.Name}}.err
`
