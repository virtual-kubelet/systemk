package systemd

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
)

func memory() string {
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

func cpu() string {
	return "4"
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func kernel() string {
	cmd := exec.Command("uname", "-r")
	buf, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(buf[:len(buf)-1])
}

func image() string {
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

func version() string {
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
