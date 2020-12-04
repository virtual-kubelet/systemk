package manager

import (
	"github.com/miekg/vks/pkg/unit"
)

// TestManager is a manager used for testing.
type TestManager struct {
	units map[string]string
}

func NewTest() (*TestManager, error) { return &TestManager{units: make(map[string]string)}, nil }

func (t *TestManager) Load(name string, u unit.File) error {
	t.units[name] = u.String()
	return nil
}

func (t *TestManager) Unload(name string) error {
	delete(t.units, name)
	return nil
}

func (t *TestManager) Mask(name string) error                       { return nil }
func (t *TestManager) TriggerStart(name string) error               { return nil }
func (t *TestManager) TriggerStop(name string) error                { return nil }
func (t *TestManager) State(name string) (*unit.State, error)       { return &unit.State{}, nil }
func (t *TestManager) Property(name, property string) string        { return "" }
func (t *TestManager) ServiceProperty(name, property string) string { return "" }
func (t *TestManager) Reload() error                                { return nil }

func (t *TestManager) Properties(name string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (t *TestManager) Unit(name string) string { return t.units[name] }
func (t *TestManager) Units() ([]string, error) {
	units := []string{}
	for k := range t.units {
		units = append(units, k)
	}
	return units, nil
}

func (t *TestManager) States(prefix string) (map[string]*unit.State, error) {
	states := make(map[string]*unit.State)
	return states, nil
}

func (t *TestManager) Disable(name string) error {
	delete(t.units, name)
	return nil
}
