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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/miekg/go-systemd/unit"
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
// Files are linked to Units by the Hash of their contents.
// Similar to systemd, a File configuration has no inherent name, but is rather
// named through the reference to it; in the case of systemd, the reference is
// the filename, and in the case of fleet, the reference is the name of the Unit
// that references this File.
type File struct {
	// Contents represents the parsed unit file.
	// This field must be considered readonly.
	Contents map[string]map[string][]string

	Options []*unit.UnitOption
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

// Hash returns the SHA1 hash of the raw contents of the Unit
func (u *File) Hash() Hash {
	return Hash(sha1.Sum(u.bytes()))
}

// MatchFiles compares two unitFiles
// Returns true if the units match, false otherwise.
func MatchFiles(a *File, b *File) bool {
	if a.Hash() == b.Hash() {
		return true
	}

	return false
}

// RecognizedUnitType determines whether or not the given unit name represents
// a recognized unit type.
func RecognizedUnitType(name string) bool {
	types := []string{"service", "socket", "timer", "path", "device", "mount", "automount"}
	for _, t := range types {
		suffix := fmt.Sprintf(".%s", t)
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// DefaultUnitType appends the default unit type to a given unit name, ignoring
// any file extensions that already exist.
func DefaultUnitType(name string) string {
	return fmt.Sprintf("%s%s", name, ServiceSuffix)
}

// ServiceSuffix is the suffix for service files. This includes the dot.
const ServiceSuffix = ".service"

// Hash is the SHa1 sum of the unit file.
type Hash [sha1.Size]byte

func (h Hash) String() string {
	return fmt.Sprintf("%x", h[:])
}

// Short returns the short hash (first 8 characters).
func (h Hash) Short() string {
	return fmt.Sprintf("%.7s", h)
}

// Empty returns true if h is the empty hash.
func (h *Hash) Empty() bool {
	return *h == Hash{}
}

func hashFromHexString(key string) (Hash, error) {
	h := Hash{}
	out, err := hex.DecodeString(key)
	if err != nil {
		return h, err
	}
	if len(out) != sha1.Size {
		return h, fmt.Errorf("size of key %q (%d) differs from SHA1 size (%d)", out, len(out), sha1.Size)
	}
	copy(h[:], out[:sha1.Size])
	return h, nil
}

// State encodes the current state of a unit loaded into a vks agent
type State struct {
	LoadState   string
	ActiveState string
	SubState    string
	MachineID   string
	UnitHash    string
	UnitName    string
	UnitData    string // the unit file as written to disk
}

// NameInfo exposes certain interesting items about a Unit based on its
// name. For example, a unit with the name "foo@.service" constitutes a
// template unit, and a unit named "foo@1.service" would represent an instance
// unit of that template.
type NameInfo struct {
	FullName string // Original complete name of the unit (e.g. foo.socket, foo@bar.service)
	Name     string // Name of the unit without suffix (e.g. foo, foo@bar)
	Prefix   string // Prefix of the template unit (e.g. foo)

	// If the unit represents an instance or a template, the following values are set
	Template string // Name of the canonical template unit (e.g. foo@.service)
	Instance string // Instance name (e.g. bar)
}

// IsInstance returns a boolean indicating whether the NameInfo appears to be
// an Instance of a Template unit
func (nu NameInfo) IsInstance() bool {
	return len(nu.Instance) > 0
}

// IsTemplate returns a boolean indicating whether the NameInfo appears to be
// a Template unit
func (nu NameInfo) IsTemplate() bool {
	return len(nu.Template) > 0 && !nu.IsInstance()
}

// NewNameInfo generates a nitNameInfo from the given name. If the given string
// is not a correct unit name, nil is returned.
func NewNameInfo(un string) *NameInfo {

	// Everything past the first @ and before the last . is the instance
	s := strings.LastIndex(un, ".")
	if s == -1 {
		return nil
	}

	nu := &NameInfo{FullName: un}
	name := un[:s]
	suffix := un[s:]
	nu.Name = name

	a := strings.Index(name, "@")
	if a == -1 {
		// This does not appear to be a template or instance unit.
		nu.Prefix = name
		return nu
	}

	nu.Prefix = name[:a]
	nu.Template = fmt.Sprintf("%s@%s", name[:a], suffix)
	nu.Instance = name[a+1:]
	return nu
}
