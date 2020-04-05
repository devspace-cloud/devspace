package testing

import (
	"fmt"

	"github.com/devspace-cloud/devspace/pkg/util/survey"
	fakesurvey "github.com/devspace-cloud/devspace/pkg/util/survey/testing"
	"github.com/sirupsen/logrus"
)

// FakeLogger just discards every log statement
type FakeLogger struct {
	Survey *fakesurvey.FakeSurvey
	Level  logrus.Level
}

// NewFakeLogger returns a new fake logger
func NewFakeLogger() *FakeLogger {
	return &FakeLogger{
		Survey: fakesurvey.NewFakeSurvey(),
	}
}

// Debug implements logger interface
func (d *FakeLogger) Debug(args ...interface{}) {}

// Debugf implements logger interface
func (d *FakeLogger) Debugf(format string, args ...interface{}) {}

// Info implements logger interface
func (d *FakeLogger) Info(args ...interface{}) {}

// Infof implements logger interface
func (d *FakeLogger) Infof(format string, args ...interface{}) {}

// Warn implements logger interface
func (d *FakeLogger) Warn(args ...interface{}) {}

// Warnf implements logger interface
func (d *FakeLogger) Warnf(format string, args ...interface{}) {}

// Error implements logger interface
func (d *FakeLogger) Error(args ...interface{}) {}

// Errorf implements logger interface
func (d *FakeLogger) Errorf(format string, args ...interface{}) {}

// Fatal implements logger interface
func (d *FakeLogger) Fatal(args ...interface{}) {
	d.Panic(args...)
}

// Fatalf implements logger interface
func (d *FakeLogger) Fatalf(format string, args ...interface{}) {
	d.Panicf(format, args...)
}

// Panic implements logger interface
func (d *FakeLogger) Panic(args ...interface{}) {
	panic(fmt.Sprint(args...))
}

// Panicf implements logger interface
func (d *FakeLogger) Panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

// Done implements logger interface
func (d *FakeLogger) Done(args ...interface{}) {}

// Donef implements logger interface
func (d *FakeLogger) Donef(format string, args ...interface{}) {}

// Fail implements logger interface
func (d *FakeLogger) Fail(args ...interface{}) {}

// Failf implements logger interface
func (d *FakeLogger) Failf(format string, args ...interface{}) {}

// Print implements logger interface
func (d *FakeLogger) Print(level logrus.Level, args ...interface{}) {}

// Printf implements logger interface
func (d *FakeLogger) Printf(level logrus.Level, format string, args ...interface{}) {}

// StartWait implements logger interface
func (d *FakeLogger) StartWait(message string) {}

// StopWait implements logger interface
func (d *FakeLogger) StopWait() {}

// SetLevel implements logger interface
func (d *FakeLogger) SetLevel(level logrus.Level) {}

// GetLevel implements logger interface
func (d *FakeLogger) GetLevel() logrus.Level { return d.Level }

// Write implements logger interface
func (d *FakeLogger) Write(message []byte) (int, error) {
	return len(message), nil
}

// WriteString implements logger interface
func (d *FakeLogger) WriteString(message string) {}

// Question asks a new question
func (d *FakeLogger) Question(params *survey.QuestionOptions) (string, error) {
	return d.Survey.Question(params)
}
