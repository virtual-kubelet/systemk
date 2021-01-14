// Copyright 2014 The fleet Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Modified for use here by Miek Gieben.

package manager

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/virtual-kubelet/systemk/pkg/unit"
	vklog "github.com/virtual-kubelet/virtual-kubelet/log"
)

// unitManager manages units via a dBus connection to systemd.
type unitManager struct {
	systemd  *dbus.Conn
	unitsDir string

	mutex sync.RWMutex
}

// Ensure Manager is fully implemented during compile time.
var _ Manager = (*unitManager)(nil)

// New returns a pointer to an initialized unitManager.
func New(uDir string, systemdUser bool) (*unitManager, error) {
	var systemd *dbus.Conn
	var err error
	if systemdUser {
		systemd, err = dbus.NewUserConnection()
	} else {
		systemd, err = dbus.New()
	}
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(uDir, os.FileMode(0755)); err != nil {
		return nil, err
	}

	mgr := unitManager{
		systemd:  systemd,
		unitsDir: uDir,
		mutex:    sync.RWMutex{},
	}
	return &mgr, nil
}

// Load writes the given Unit to disk, subscribing to relevant dbus
// events.
func (m *unitManager) Load(name string, u unit.File) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	err := m.writeUnit(name, u.String())
	if err != nil {
		return err
	}
	if _, exists := u.Contents["Install"]; exists {
		ok, err := m.enableUnit(name)
		if err != nil || !ok {
			m.removeUnit(name)
			return fmt.Errorf("Failed to enable systemd unit %s: %v", name, err)
		}
	}
	return nil
}

// Unload removes the indicated unit from the filesystem, and clears its unit status in systemd
func (m *unitManager) Unload(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.removeUnit(name)
}

// TriggerStart asynchronously starts the unit identified by the given name.
// This function does not block for the underlying unit to actually start.
func (m *unitManager) TriggerStart(name string) error {
	jobID, err := m.systemd.StartUnit(name, "replace", nil)
	if err != nil {
		return err
	}
	vklog.G(context.TODO()).Infof("Triggered systemd unit %s start: job=%d", name, jobID)
	return nil
}

// TriggerStop asynchronously starts the unit identified by the given name.
// This function does not block for the underlying unit to actually stop.
func (m *unitManager) TriggerStop(name string) error {
	_, err := m.systemd.StopUnit(name, "replace", nil)
	if err != nil {
		return err
	}
	return nil
}

// State generates a State object representing the
// current state of a Unit
func (m *unitManager) State(name string) (*unit.State, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	us, err := m.getState(name)
	if err != nil {
		return nil, err
	}
	return us, nil
}

func (m *unitManager) getState(name string) (*unit.State, error) {
	info, err := m.systemd.GetUnitProperties(name)
	if err != nil {
		return nil, err
	}
	us := &unit.State{UnitStatus: dbus.UnitStatus{
		Description: info["Description"].(string),
		LoadState:   info["LoadState"].(string),
		ActiveState: info["ActiveState"].(string),
		SubState:    info["SubState"].(string),
	}}
	return us, nil
}

// Properties returns the properties of the unit.
// Probably need the service properpty, not the unit.
func (m *unitManager) Properties(name string) (map[string]interface{}, error) {
	return m.systemd.GetUnitProperties(name)
}

// Property returns the property of the unit.
func (m *unitManager) Property(name, property string) string {
	p, err := m.systemd.GetUnitProperty(name, property)
	if err != nil {
		return ""
	}
	if p == nil {
		return ""
	}
	return p.Value.String()
}

// ServiceProperty returns the property of the named unit.
func (m *unitManager) ServiceProperty(name, property string) string {
	p, err := m.systemd.GetServiceProperty(name, property)
	if err != nil {
		return ""
	}
	if p == nil {
		return ""
	}
	// these value string encode the type with @<Char><Space>, if so remove it before returning
	vs := p.Value.String()
	if vs[0] == '@' {
		return vs[3:]
	}
	return vs
}

func (m *unitManager) readUnit(name string) (string, error) {
	path := m.getUnitFilePath(name)
	contents, err := ioutil.ReadFile(path)
	if err == nil {
		return string(contents), nil
	}
	return "", fmt.Errorf("no unit file at local path %s", path)
}

// Reload tells systemd to reload all unit files.
func (m *unitManager) Reload() error { return m.systemd.Reload() }

// Unit returns the named unit from the manager. This is used in testing only.
func (m *unitManager) Unit(name string) string {
	vklog.G(context.TODO()).Info("not implemented, used for testing")
	return ""
}

// Units enumerates all files recognized as valid systemd units in
// this manager's units directory.
func (m *unitManager) Units() ([]string, error) { return lsUnitsDir(m.unitsDir) }

// States return all units the have the prefix "systemk."
func (m *unitManager) States(prefix string) (map[string]*unit.State, error) {
	dbusStatuses, err := m.systemd.ListUnits()
	if err != nil {
		return nil, err
	}
	vklog.G(context.TODO()).Infof("%d statuses returned", len(dbusStatuses))

	states := make(map[string]*unit.State)
	for _, dus := range dbusStatuses {
		if !strings.HasPrefix(dus.Name, prefix) {
			continue
		}
		if !strings.HasSuffix(dus.Name, ".service") {
			continue
		}
		us := &unit.State{UnitStatus: dus}
		if buf, err := m.readUnit(dus.Name); err == nil {
			// this should not error, but ... TODO(miek)
			us.UnitData = buf
		}
		states[dus.Name] = us
	}

	vklog.G(context.TODO()).Infof("Left with %d statuses for prefix %q", len(states), prefix)

	return states, nil
}

func (m *unitManager) writeUnit(name string, contents string) error {
	bContents := []byte(contents)
	vklog.G(context.TODO()).Infof("Writing systemd unit %s (%db)", name, len(bContents))

	ufPath := m.getUnitFilePath(name)
	err := ioutil.WriteFile(ufPath, bContents, os.FileMode(0644))
	if err != nil {
		return err
	}

	_, err = m.systemd.LinkUnitFiles([]string{ufPath}, true, true)
	return err
}

func (m *unitManager) enableUnit(name string) (bool, error) {
	vklog.G(context.TODO()).Infof("Enabling systemd unit %s", name)

	ufPath := m.getUnitFilePath(name)

	ok, _, err := m.systemd.EnableUnitFiles([]string{ufPath}, true, true)
	return ok, err
}

func (m *unitManager) removeUnit(name string) (err error) {
	vklog.G(context.TODO()).Infof("Removing systemd unit %s", name)

	// both DisableUnitFiles() and ResetFailedUnit() must be followed by
	// removing the unit file. Otherwise "systemctl stop fleet" could end up hanging forever.
	if errf := m.Disable(name); errf != nil {
		err = fmt.Errorf("%v, %v", err, errf)
	}

	if errf := m.systemd.ResetFailedUnit(name); errf != nil {
		err = fmt.Errorf("%v, %v", err, errf)
	}

	ufPath := m.getUnitFilePath(name)
	os.Remove(ufPath)

	return err
}

// Disable disable the unit named via name.
func (m *unitManager) Disable(name string) error {
	_, err := m.systemd.DisableUnitFiles([]string{name}, true)
	return err
}

// Mask masks a unit.
func (m *unitManager) Mask(name string) error {
	_, err := m.systemd.MaskUnitFiles([]string{name}, true, true)
	return err
}

func (m *unitManager) getUnitFilePath(name string) string {
	return path.Join(m.unitsDir, name)
}

func lsUnitsDir(dir string) ([]string, error) {
	filterFunc := func(name string) bool {
		if !strings.HasSuffix(name, unit.ServiceSuffix) {
			vklog.G(context.TODO()).Infof("Found unrecognized file in %s, ignoring", path.Join(dir, name))
			return true
		}

		return false
	}

	return listDirectory(dir, filterFunc)
}

// listDirectory generates a slice of all the file names that both exist in
// the provided directory and pass the filter.
// The returned file names are relative to the directory argument.
// filterFunc is called once for each file found in the directory. If
// filterFunc returns true, the given file will ignored.
func listDirectory(dir string, filterFunc func(string) bool) ([]string, error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	units := make([]string, 0)
	for _, fi := range fis {
		name := fi.Name()
		if filterFunc(name) {
			continue
		}
		units = append(units, name)
	}

	return units, nil
}
