package systemd

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type ProbeStatus struct {
	m map[string]bool // instead of a bool this probably needs to be a tri-bool?
	c map[string]chan struct{}
	sync.RWMutex
}

// status returns the status for name. Name consists out of
// namespace "." name "." container.
func (ps *ProbeStatus) status(name string) bool {
	ps.RLock()
	defer ps.RUnlock()
	return ps.m[name]
}

// setStatus sets the status for name to status.
func (ps *ProbeStatus) setStatus(name string, status bool) {
	ps.Lock()
	defer ps.Unlock()
	ps.m[name] = status
}

// stop stops the probes for name.
func (ps *ProbeStatus) stop(name string) {
	ps.RLock()
	defer ps.RUnlock()
	ch := ps.c[name]
	if ch == nil {
		return
	}
	close(ch)
}

// setStop sets the stop channel for name.
func (ps *ProbeStatus) setStop(name string, ch chan struct{}) {
	ps.Lock()
	defer ps.Unlock()
	ps.c[name] = ch
}

// clean removes all traces of name from the probestatus.
func (ps *ProbeStatus) Clean(name string) {
	ps.Lock()
	defer ps.Unlock()
	delete(ps.m, name)
	delete(ps.c, name)
}

func (h *httpGet) Do(url string) bool {
	c := new(http.Client)
	c.Timeout = h.timeout
	// use h.headers here

	resp, err := c.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
		return true
	}
	return false
}

func (h *httpGet) Settings() settings { return h.settings }

func (t *tcpSocket) Do(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, t.timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func (t *tcpSocket) Settings() settings { return t.settings }

type settings struct {
	delay            time.Duration
	period           time.Duration
	timeout          time.Duration
	successThreshold int
	failureThreshold int
}

func newDefaultSettings() settings {
	return settings{
		period:           10 * time.Second,
		successThreshold: 1,
		failureThreshold: 3,
	}
}

type httpGet struct {
	settings
	headers http.Header
}

type tcpSocket struct {
	settings
}

type prober interface {
	Do(string) bool
	Settings() settings
}

func (ps *ProbeStatus) Probe(name, what string, p prober) chan struct{} {
	stop := make(chan struct{})
	go func() {
		time.Sleep(p.Settings().delay)
		tick := time.NewTicker(p.Settings().period)
		defer tick.Stop()
		ok := 0
		fail := 0
		for {
			select {
			case <-tick.C:
				stat := p.Do(what)
				if stat {
					ok++
					fail = 0
				} else {
					ok = 0
					fail++
				}

				if ok >= p.Settings().successThreshold {
					ps.setStatus(name, true)
				}
				if fail >= p.Settings().failureThreshold {
					ps.setStatus(name, false)
				}
			case <-stop:
				return
			}
		}
	}()

	return stop
}
