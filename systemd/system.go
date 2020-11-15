package systemd

import (
	"bytes"
	"fmt"
	"io/ioutil"
)

func memory() (string, error) {
	buf, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return "", err
	}
	// First line is MemTotal which we're intested in
	i := bytes.Index(buf, []byte("\n"))
	if i == 0 {
		return "", fmt.Errorf("")
	}
	line := bytes.ReplaceAll(buf[:i], []byte("MemTotal:"), []byte{})
	return string(bytes.TrimSpace(line)), nil
}

func cpu() (string, error) {
	return "", nil
}
