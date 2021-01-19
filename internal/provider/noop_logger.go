package provider

import (
	vklog "github.com/virtual-kubelet/virtual-kubelet/log"
)

// noopLogger is useful in tests.
type noopLogger struct{}

// Ensure Logger implementation.
var _ vklog.Logger = (*noopLogger)(nil)

func (e *noopLogger) Debug(i ...interface{}) {}

func (e *noopLogger) Debugf(s string, i ...interface{}) {}

func (e *noopLogger) Info(i ...interface{}) {}

func (e *noopLogger) Infof(s string, i ...interface{}) {}

func (e *noopLogger) Warn(i ...interface{}) {}

func (e *noopLogger) Warnf(s string, i ...interface{}) {}

func (e *noopLogger) Error(i ...interface{}) {}

func (e *noopLogger) Errorf(s string, i ...interface{}) {}

func (e *noopLogger) Fatal(i ...interface{}) {}

func (e *noopLogger) Fatalf(s string, i ...interface{}) {}

func (e *noopLogger) WithField(s string, i interface{}) vklog.Logger { return e }

func (e *noopLogger) WithFields(fields vklog.Fields) vklog.Logger { return e }

func (e *noopLogger) WithError(err error) vklog.Logger { return e }
