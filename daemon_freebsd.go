package daemon

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type DaemonService struct {
	ServiceProperties
}

// GetTemplate - gets service config template
func (svc *DaemonService) GetTemplate() string {
	return bsdConfig
}

// SetTemplate - sets service config template
func (svc *DaemonService) SetTemplate(tplStr string) error {
	bsdConfig = tplStr
	return nil
}

// Standard service path for systemV daemons
func (svc *DaemonService) servicePath() string {
	return "/usr/local/etc/rc.d/" + svc.name
}

// Is a service installed
func (svc *DaemonService) isInstalled() bool {
	if _, err := os.Stat(svc.servicePath()); err == nil {
		return true
	}

	return false
}

// Is a service is enabled
func (svc *DaemonService) isEnabled() (bool, error) {
	rcConf, err := os.Open("/etc/rc.conf")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return false, err
	}
	defer rcConf.Close()
	rcData, _ := ioutil.ReadAll(rcConf)
	r, _ := regexp.Compile(`.*` + svc.name + `_enable="YES".*`)
	v := string(r.Find(rcData))
	var chrFound, sharpFound bool
	for _, c := range v {
		if c == '#' && !chrFound {
			sharpFound = true
			break
		} else if !sharpFound && c != ' ' {
			chrFound = true
			break
		}
	}
	return chrFound, nil
}

func (svc *DaemonService) getCmd(cmd string) string {
	if ok, err := svc.isEnabled(); !ok || err != nil {
		fmt.Println("Service is not enabled, using one" + cmd + " instead")
		cmd = "one" + cmd
	}
	return cmd
}

// Get the daemon properly
func newDaemon(name, description string, dependencies []string) (Daemon, error) {
	return &Service{name, description, dependencies}, nil
}

func execPath() (name string, err error) {
	name = os.Args[0]
	if name[0] == '.' {
		name, err = filepath.Abs(name)
		if err == nil {
			name = filepath.Clean(name)
		}
	} else {
		name, err = exec.LookPath(filepath.Clean(name))
	}
	return name, err
}

// Check service is running
func (svc *DaemonService) checkRunning() (string, bool) {
	output, err := exec.Command("service", svc.name, svc.getCmd("status")).Output()
	if err == nil {
		if matched, err := regexp.MatchString(svc.name, string(output)); err == nil && matched {
			reg := regexp.MustCompile("pid  ([0-9]+)")
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
func (svc *DaemonService) Install(args ...string) (string, error) {
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

	templ, err := template.New("bsdConfig").Parse(bsdConfig)
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
func (svc *DaemonService) Remove() (string, error) {
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
func (svc *DaemonService) Start() (string, error) {
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

	if err := exec.Command("service", svc.name, svc.getCmd("start")).Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (svc *DaemonService) Stop() (string, error) {
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

	if err := exec.Command("service", svc.name, svc.getCmd("stop")).Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (svc *DaemonService) Status() (string, error) {

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
func (svc *DaemonService) Run(e Executable) (string, error) {
	runAction := "Running " + svc.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

var bsdConfig = `#!/bin/sh
#
# PROVIDE: {{.Name}}
# REQUIRE: networking syslog
# KEYWORD:

# Add the following lines to /etc/rc.conf to enable the {{.Name}}:
#
# {{.Name}}_enable="YES"
#


. /etc/rc.subr

name="{{.Name}}"
rcvar="{{.Name}}_enable"
command="{{.Path}}"
pidfile="/var/run/$name.pid"

start_cmd="/usr/sbin/daemon -p $pidfile -f $command {{.Args}}"
load_rc_config $name
run_rc_command "$1"
`
