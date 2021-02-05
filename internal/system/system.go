package system

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

var (
	// this is a variable so it can be overridden during unit-testing.
	osReleaseFilePath = "/etc/os-release"
)

// Memory returns the amount of memory in the system.
func Memory() string {
	in := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(in)
	if err != nil {
		return ""
	}

	// If this is a 32-bit system, then these fields are
	// uint32 instead of uint64.
	// So we always convert to uint64 to match signature.
	// We divide / 1024 since we want kb

	totalMem := uint64(in.Totalram) * uint64(in.Unit) / 1024
	str := fmt.Sprintf("%sk", strconv.FormatUint(totalMem, 10))
	return str
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
	buf, err := ioutil.ReadFile(osReleaseFilePath)
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
	// Some attributes are quoted, some are not. Cover both.
	os = bytes.ReplaceAll(os[:j-1], []byte("\""), []byte{})
	return string(os)
}

// Version returns the version of systemd.
func Version() string {
	cmd := exec.Command("systemctl", "--version")
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
	buf, err := ioutil.ReadFile(osReleaseFilePath)
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
	// Some attributes are quoted, some are not. Cover both.
	id = bytes.ReplaceAll(id[:j], []byte("\""), []byte{})
	return string(id)
}

// Pid returns the PID space / 4.
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

// IPs returns the IP addresses found on the interfaces.
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

// IPFromInterface returns the first address found on the interface named name.
func IPFromInterface(name string) (net.IP, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("failed to find addresses on interface %q", name)
	}
	switch v := addrs[0].(type) {
	case *net.IPNet:
		return v.IP, nil
	case *net.IPAddr:
		return v.IP, nil
	}
	return nil, fmt.Errorf("neither net or address found on interface %q", name)
}
