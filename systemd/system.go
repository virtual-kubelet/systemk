package systemd

import (
	"bytes"
	"io/ioutil"
	"os"
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
