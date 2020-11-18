package system

import (
	"bytes"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
)

// Memory returns the amount of memory in the system.
func Memory() string {
	buf, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return ""
	}
	// First line is MemTotal which we're intested in
	i := bytes.Index(buf, []byte("\n"))
	if i == 0 {
		return ""
	}
	line := bytes.ReplaceAll(buf[:i], []byte("MemTotal:"), []byte{})
	line = bytes.TrimSpace(line)
	line = bytes.ReplaceAll(line, []byte(" "), []byte{}) // space between number and unit
	amount := line[:len(line)-1]                         // cut of last B
	return string(amount)
}

// CPU returns the number of CPUs in the system as reported by nproc.
func CPU() string {
	cmd := exec.Command("nproc")
	buf, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(buf[:len(buf)-1])
}

// Hostname returns the machine's host name.
func Hostname() string {
	h, _ := os.Hostname()
	return h
}

// Kernel returns kernel version.
func Kernel() string {
	cmd := exec.Command("uname", "-r")
	buf, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(buf[:len(buf)-1])
}

// Image returns the systems image (PRETTY_NAME from /etc/os-release)
func Image() string {
	buf, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	i := bytes.Index(buf, []byte("PRETTY_NAME="))
	if i == 0 {
		return ""
	}
	os := buf[i+len("PRETTY_NAME="):]
	j := bytes.Index(os, []byte("\n"))
	if j == 0 {
		return ""
	}
	os = os[:j]
	os = bytes.ReplaceAll(os, []byte("\""), []byte{})
	return string(os[:len(os)-1]) // newline
}

// Version returns the version of systemd.
func Version() string {
	cmd := exec.Command("systemd", "--version")
	buf, err := cmd.Output()
	if err != nil {
		return ""
	}
	i := bytes.Index(buf, []byte("\n"))
	if i == 0 {
		return ""
	}
	return string(buf[:i])
}

// ID returns the ID of the system.
func ID() string {
	buf, err := ioutil.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	i := bytes.Index(buf, []byte("ID="))
	if i == 0 {
		return ""
	}
	id := buf[i+len("ID="):]
	j := bytes.Index(id, []byte("\n"))
	if j == 0 {
		return ""
	}
	id = id[:j]
	return string(id)
}

// Returns the PID space / 4 (/4 is random)
func Pid() string {
	buf, err := ioutil.ReadFile("/proc/sys/kernel/pid_max")
	if err != nil || len(buf) < 2 {
		return ""
	}
	buf = buf[:len(buf)-1] // strip newline
	pid, err := strconv.Atoi(string(buf))
	if err != nil {
		return ""
	}
	pid = pid / 4
	return strconv.Itoa(pid)
}

func IPs() []net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	a := []net.IP{}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				continue
			}
			a = append(a, ip)
		}
	}
	return a
}
