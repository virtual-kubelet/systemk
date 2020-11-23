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
	"reflect"
	"testing"
)

func TestDefaultUnitType(t *testing.T) {
	tts := []struct {
		name string
		out  string
	}{
		{"foo", "foo.service"},
		{"foo.service", "foo.service.service"},
		{"foo.link", "foo.link.service"},
	}

	for _, tt := range tts {
		out := DefaultUnitType(tt.name)
		if out != tt.out {
			t.Errorf("Case failed: name=%s expect=%s result=%s", tt.name, tt.out, out)
		}
	}
}

func TestDeserialize(t *testing.T) {
	contents := `
This=Ignored
[Unit]
;ignore this guy
Description = Foo

[Service]
ExecStart=echo "ping";
ExecStop=echo "pong"
# ignore me, too
ExecStop=echo post

[X-Fleet]
MachineMetadata=foo=bar
MachineMetadata=baz=qux
`

	expected := map[string]map[string][]string{
		"Unit": {
			"Description": {"Foo"},
		},
		"Service": {
			"ExecStart": {`echo "ping";`},
			"ExecStop":  {`echo "pong"`, "echo post"},
		},
		"X-Fleet": {
			"MachineMetadata": {"foo=bar", "baz=qux"},
		},
	}

	unitFile, err := New(contents)
	if err != nil {
		t.Fatalf("Unexpected error parsing unit %q: %v", contents, err)
	}

	if !reflect.DeepEqual(expected, unitFile.Contents) {
		t.Fatalf("Map func did not produce expected output.\nActual=%v\nExpected=%v", unitFile.Contents, expected)
	}
}

func TestDeserializedUnitGarbage(t *testing.T) {
	contents := `
>>>>>>>>>>>>>
[Service]
ExecStart=jim
# As long as a line has an equals sign, systemd is happy, so we should pass it through
<<<<<<<<<<<=bar
`
	expected := map[string]map[string][]string{
		"Service": {
			"ExecStart":   {"jim"},
			"<<<<<<<<<<<": {"bar"},
		},
	}
	unitFile, err := New(contents)
	if err != nil {
		t.Fatalf("Unexpected error parsing unit %q: %v", contents, err)
	}

	if !reflect.DeepEqual(expected, unitFile.Contents) {
		t.Fatalf("Map func did not produce expected output.\nActual=%v\nExpected=%v", unitFile.Contents, expected)
	}
}

func TestDeserializeEscapedMultilines(t *testing.T) {
	contents := `
[Service]
ExecStart=echo \
  "pi\
  ng"
ExecStop=\
echo "po\
ng"
# comments within continuation should not be ignored
ExecStopPre=echo\
#pang
ExecStopPost=echo\
#peng\
pung
`
	expected := map[string]map[string][]string{
		"Service": {
			"ExecStart": {`echo \
  "pi\
  ng"`},
			"ExecStop": {`\
echo "po\
ng"`},
			"ExecStopPre": {`echo\
#pang`},
			"ExecStopPost": {`echo\
#peng\
pung`},
		},
	}
	unitFile, err := New(contents)
	if err != nil {
		t.Fatalf("Unexpected error parsing unit %q: %v", contents, err)
	}

	if !reflect.DeepEqual(expected, unitFile.Contents) {
		t.Fatalf("Map func did not produce expected output.\nActual=%v\nExpected=%v", unitFile.Contents, expected)
	}
}

func TestSerializeDeserialize(t *testing.T) {
	contents := `
[Unit]
Description = Foo
`
	deserialized, err := New(contents)
	if err != nil {
		t.Fatalf("Unexpected error parsing unit %q: %v", contents, err)
	}
	section := deserialized.Contents["Unit"]
	if val, ok := section["Description"]; !ok || val[0] != "Foo" {
		t.Errorf("Failed to persist data through serialize/deserialize: %v", val)
	}

	serialized := deserialized.String()
	deserialized, err = New(serialized)
	if err != nil {
		t.Fatalf("Unexpected error parsing unit %q: %v", serialized, err)
	}

	section = deserialized.Contents["Unit"]
	if val, ok := section["Description"]; !ok || val[0] != "Foo" {
		t.Errorf("Failed to persist data through serialize/deserialize: %v", val)
	}
}

func TestDescription(t *testing.T) {
	contents := `
[Unit]
Description = Foo

[Service]
ExecStart=echo "ping";
ExecStop=echo "pong";
`

	unitFile, err := New(contents)
	if err != nil {
		t.Fatalf("Unexpected error parsing unit %q: %v", contents, err)
	}
	if unitFile.Description() != "Foo" {
		t.Fatalf("Unit.Description is incorrect")
	}
}

func TestDescriptionNotDefined(t *testing.T) {
	contents := `
[Unit]

[Service]
ExecStart=echo "ping";
ExecStop=echo "pong";
`

	unitFile, err := New(contents)
	if err != nil {
		t.Fatalf("Unexpected error parsing unit %q: %v", contents, err)
	}
	if unitFile.Description() != "" {
		t.Fatalf("Unit.Description is incorrect")
	}
}

func TestBadUnitsFail(t *testing.T) {
	bad := []string{
		`
[Unit]

[Service]
<<<<<<<<<<<<<<<<
`,
		`
[Unit]
nonsense upon stilts
`,
	}
	for _, tt := range bad {
		if _, err := New(tt); err == nil {
			t.Fatalf("Did not get expected error creating Unit from %q", tt)
		}
	}
}

func TestInsert(t *testing.T) {
	contents := `
[Unit]
Description = Foo

[Service]
ExecStart=echo "ping";
ExecStop=echo "pong";
`

	unitFile, _ := New(contents)
	unitFile = unitFile.Insert("bla", "bloep", "myvalue")

	if x := unitFile.Contents["bla"]["bloep"]; x[0] != "myvalue" {
		t.Fatalf("expected %s, got %s", "myvalue", x[0])
	}
}
