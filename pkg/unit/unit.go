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

package unit

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/coreos/go-systemd/v22/unit"
)

// New returns a new unit file parsed from raw.
func New(raw string) (*File, error) {
	reader := strings.NewReader(raw)
	opts, err := unit.Deserialize(reader)
	if err != nil {
		return nil, err
	}

	return newFromOptions(opts), nil
}

// newFromOptions returns a new unit file parsed from opts.
func newFromOptions(opts []*unit.UnitOption) *File {
	return &File{mapOptions(opts), opts}
}

func mapOptions(opts []*unit.UnitOption) map[string]map[string][]string {
	contents := make(map[string]map[string][]string)
	for _, opt := range opts {
		if _, ok := contents[opt.Section]; !ok {
			contents[opt.Section] = make(map[string][]string)
		}

		if _, ok := contents[opt.Section][opt.Name]; !ok {
			contents[opt.Section][opt.Name] = make([]string, 0)
		}

		contents[opt.Section][opt.Name] = append(contents[opt.Section][opt.Name], opt.Value)
	}

	return contents
}

// A File represents a systemd configuration which encodes information about any of the unit
// types that fleet supports (as defined in SupportedUnitTypes()).
type File struct {
	Contents map[string]map[string][]string
	Options  []*unit.UnitOption
}

// Description returns the first Description option found in the [Unit] section.
// If the option is not defined, an empty string is returned.
func (u *File) Description() string {
	if values := u.Contents["Unit"]["Description"]; len(values) > 0 {
		return values[0]
	}
	return ""
}

func (u *File) bytes() []byte {
	b, _ := ioutil.ReadAll(unit.Serialize(u.Options))
	return b
}

func (u *File) String() string {
	return string(u.bytes())
}

// Insert adds name=value to section and returns a newly parsed pointer to File.
func (u *File) Insert(section, name string, value ...string) *File {
	opts := make([]*unit.UnitOption, len(value))
	for i := range opts {
		opts[i] = &unit.UnitOption{
			Section: section,
			Name:    name,
			Value:   value[i],
		}
	}
	u.Options = append(u.Options, opts...)
	return newFromOptions(u.Options)
}

// Overwrite overwrites name=value in the section and returns a new File.
func (u *File) Overwrite(section, name string, value ...string) *File {
	opts := make([]*unit.UnitOption, len(u.Options))
	j := 0
	for _, o := range u.Options {
		if o.Section == section && o.Name == name {
			continue
		}
		opts[j] = o
		j++
	}
	u.Options = opts[:j]
	return u.Insert(section, name, value...)
}

// Delete deletes name in the named section and returns a new File.
func (u *File) Delete(section, name string) *File {
	opts := make([]*unit.UnitOption, len(u.Options))
	j := 0
	for _, o := range u.Options {
		if o.Section == section && o.Name == name {
			continue
		}
		opts[j] = o
		j++
	}
	u.Options = opts[:j]
	return u
}

// DefaultUnitType appends the default unit type to a given unit name, ignoring
// any file extensions that already exist.
func DefaultUnitType(name string) string {
	return fmt.Sprintf("%s%s", name, ServiceSuffix)
}

const (
	// ServiceSuffix is the suffix for service files. This includes the dot.
	ServiceSuffix = ".service"
)

// State encodes the current state of a unit loaded into a systemk agent
type State struct {
	dbus.UnitStatus
	UnitData string // the unit file as written to disk
}
