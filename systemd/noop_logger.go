package systemd

import "github.com/go-logr/logr"

// noopLogger disables klogs, this is useful in tests.
type noopLogger struct{}

func (e *noopLogger) Enabled() bool                                             { return false }
func (e *noopLogger) Info(msg string, keysAndValues ...interface{})             {}
func (e *noopLogger) Error(err error, msg string, keysAndValues ...interface{}) {}
func (e *noopLogger) V(level int) logr.Logger                                   { return e }
func (e *noopLogger) WithValues(keysAndValues ...interface{}) logr.Logger       { return e }
func (e *noopLogger) WithName(name string) logr.Logger                          { return e }
