package daemon

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

type SystemvService struct {
	ServiceProperties
}

// GetTemplate - gets service config template
func (svc *SystemvService) GetTemplate() string {
	return systemVConfig
}

// SetTemplate - sets service config template
func (svc *SystemvService) SetTemplate(tplStr string) error {
	systemVConfig = tplStr
	return nil
}

// Standard service path for systemV daemons
func (svc *SystemvService) servicePath() string {
	return "/etc/init.d/" + svc.name
}

// Is a service installed
func (svc *SystemvService) isInstalled() bool {

	if _, err := os.Stat(svc.servicePath()); err == nil {
		return true
	}

	return false
}

// Check service is running
func (svc *SystemvService) checkRunning() (string, bool) {
	output, err := exec.Command("service", svc.name, "status").Output()
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
func (svc *SystemvService) Install(args ...string) (string, error) {
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

	templ, err := template.New("systemVConfig").Parse(systemVConfig)
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

	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Symlink(srvPath, "/etc/rc"+i+".d/S87"+svc.name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Symlink(srvPath, "/etc/rc"+i+".d/K17"+svc.name); err != nil {
			continue
		}
	}

	return installAction + success, nil
}

// Remove the service
func (svc *SystemvService) Remove() (string, error) {
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

	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err := os.Remove("/etc/rc" + i + ".d/S87" + svc.name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err := os.Remove("/etc/rc" + i + ".d/K17" + svc.name); err != nil {
			continue
		}
	}

	return removeAction + success, nil
}

// Start the service
func (svc *SystemvService) Start() (string, error) {
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

	if err := exec.Command("service", svc.name, "start").Run(); err != nil {
		return startAction + failed, err
	}

	return startAction + success, nil
}

// Stop the service
func (svc *SystemvService) Stop() (string, error) {
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

	if err := exec.Command("service", svc.name, "stop").Run(); err != nil {
		return stopAction + failed, err
	}

	return stopAction + success, nil
}

// Status - Get service status
func (svc *SystemvService) Status() (string, error) {

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
func (svc *SystemvService) Run(e Executable) (string, error) {
	runAction := "Running " + svc.description + ":"
	e.Run()
	return runAction + " completed.", nil
}

var systemVConfig = `#! /bin/sh
#
#       /etc/rc.d/init.d/{{.Name}}
#
#       Starts {{.Name}} as a daemon
#
# chkconfig: 2345 87 17
# description: Starts and stops a single {{.Name}} instance on this system

### BEGIN INIT INFO
# Provides: {{.Name}} 
# Required-Start: $network $named
# Required-Stop: $network $named
# Default-Start: 2 3 4 5
# Default-Stop: 0 1 6
# Short-Description: This service manages the {{.Description}}.
# Description: {{.Description}}
### END INIT INFO

#
# Source function library.
#
if [ -f /etc/rc.d/init.d/functions ]; then
    . /etc/rc.d/init.d/functions
fi

exec="{{.Path}}"
servname="{{.Description}}"

proc="{{.Name}}"
pidfile="/var/run/$proc.pid"
lockfile="/var/lock/subsys/$proc"
stdoutlog="/var/log/$proc.log"
stderrlog="/var/log/$proc.err"

[ -d $(dirname $lockfile) ] || mkdir -p $(dirname $lockfile)

[ -e /etc/sysconfig/$proc ] && . /etc/sysconfig/$proc

start() {
    [ -x $exec ] || exit 5

    if [ -f $pidfile ]; then
        if ! [ -d "/proc/$(cat $pidfile)" ]; then
            rm $pidfile
            if [ -f $lockfile ]; then
                rm $lockfile
            fi
        fi
    fi

    if ! [ -f $pidfile ]; then
        printf "Starting $servname:\t"
        echo "$(date)" >> $stdoutlog
        $exec {{.Args}} >> $stdoutlog 2>> $stderrlog &
        echo $! > $pidfile
        touch $lockfile
        success
        echo
    else
        # failure
        echo
        printf "$pidfile still exists...\n"
        exit 7
    fi
}

stop() {
    echo -n $"Stopping $servname: "
    killproc -p $pidfile $proc
    retval=$?
    echo
    [ $retval -eq 0 ] && rm -f $lockfile
    return $retval
}

restart() {
    stop
    start
}

rh_status() {
    status -p $pidfile $proc
}

rh_status_q() {
    rh_status >/dev/null 2>&1
}

case "$1" in
    start)
        rh_status_q && exit 0
        $1
        ;;
    stop)
        rh_status_q || exit 0
        $1
        ;;
    restart)
        $1
        ;;
    status)
        rh_status
        ;;
    *)
        echo $"Usage: $0 {start|stop|status|restart}"
        exit 2
esac

exit $?
`
