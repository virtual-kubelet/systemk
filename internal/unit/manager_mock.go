// Copyright © 2021 The systemk authors
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

// mockManager is a manager used for testing.
type mockManager struct {
	units map[string]string
}

// Asset mockManager fulfills Manager.
var _ Manager = (*mockManager)(nil)

func NewMockManager() (Manager, error) { return &mockManager{units: make(map[string]string)}, nil }

func (t *mockManager) Load(name string, u File) error {
	t.units[name] = u.String()
	return nil
}

func (t *mockManager) Unload(name string) error {
	delete(t.units, name)
	return nil
}

func (t *mockManager) TriggerStart(name string) error               { return nil }
func (t *mockManager) TriggerStop(name string) error                { return nil }
func (t *mockManager) State(name string) (*State, error)            { return &State{}, nil }
func (t *mockManager) Property(name, property string) string        { return "" }
func (t *mockManager) ServiceProperty(name, property string) string { return "" }
func (t *mockManager) Reload() error                                { return nil }
func (t *mockManager) Mask(name string) error                       { return nil }

func (t *mockManager) Properties(name string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (t *mockManager) Unit(name string) string { return t.units[name] }
func (t *mockManager) Units() ([]string, error) {
	units := []string{}
	for k := range t.units {
		units = append(units, k)
	}
	return units, nil
}

func (t *mockManager) States(prefix string) (map[string]*State, error) {
	states := make(map[string]*State)
	return states, nil
}

func (t *mockManager) Disable(name string) error {
	delete(t.units, name)
	return nil
}
