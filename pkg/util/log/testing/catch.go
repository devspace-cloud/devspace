package testing

import (
	"fmt"

	"github.com/loft-sh/devspace/pkg/util/survey"
	fakesurvey "github.com/loft-sh/devspace/pkg/util/survey/testing"
	"github.com/sirupsen/logrus"
)

// CatchLogger collects all logs and stores them
type CatchLogger struct {
	Survey *fakesurvey.FakeSurvey
	logs   string
}

// NewCatchLogger returns a new catch logger
func NewCatchLogger() *CatchLogger {
	return &CatchLogger{
		Survey: fakesurvey.NewFakeSurvey(),
	}
}

// GetLogs returns the logs until now
func (c *CatchLogger) GetLogs() string {
	return c.logs
}

// Debug implements logger interface
func (c *CatchLogger) Debug(args ...interface{}) {
	c.logs = c.logs + "\n[DEBUG] " + fmt.Sprint(args...)
}

// Debugf implements logger interface
func (c *CatchLogger) Debugf(format string, args ...interface{}) {
	c.logs = c.logs + "\n[DEBUG] " + fmt.Sprintf(format, args...)
}

// Info implements logger interface
func (c *CatchLogger) Info(args ...interface{}) {
	c.logs = c.logs + "\n[INFO] " + fmt.Sprint(args...)
}

// Infof implements logger interface
func (c *CatchLogger) Infof(format string, args ...interface{}) {
	c.logs = c.logs + "\n[INFO] " + fmt.Sprintf(format, args...)
}

// Warn implements logger interface
func (c *CatchLogger) Warn(args ...interface{}) {
	c.logs = c.logs + "\n[WARN] " + fmt.Sprint(args...)
}

// Warnf implements logger interface
func (c *CatchLogger) Warnf(format string, args ...interface{}) {
	c.logs = c.logs + "\n[WARN] " + fmt.Sprintf(format, args...)
}

// Error implements logger interface
func (c *CatchLogger) Error(args ...interface{}) {
	c.logs = c.logs + "\n[ERROR] " + fmt.Sprint(args...)
}

// Errorf implements logger interface
func (c *CatchLogger) Errorf(format string, args ...interface{}) {
	c.logs = c.logs + "\n[ERROR] " + fmt.Sprintf(format, args...)
}

// Fatal implements logger interface
func (c *CatchLogger) Fatal(args ...interface{}) {
	c.logs = c.logs + "\n[FATAL] " + fmt.Sprint(args...)
	c.Panic(args...)
}

// Fatalf implements logger interface
func (c *CatchLogger) Fatalf(format string, args ...interface{}) {
	c.logs = c.logs + "\n[FATAL] " + fmt.Sprintf(format, args...)
	c.Panicf(format, args...)
}

// Panic implements logger interface
func (c *CatchLogger) Panic(args ...interface{}) {
	c.logs = c.logs + "\n[PANIC] " + fmt.Sprint(args...)
	panic(fmt.Sprint(args...))
}

// Panicf implements logger interface
func (c *CatchLogger) Panicf(format string, args ...interface{}) {
	c.logs = c.logs + "\n[PANIC] " + fmt.Sprintf(format, args...)
	panic(fmt.Sprintf(format, args...))
}

// Done implements logger interface
func (c *CatchLogger) Done(args ...interface{}) {
	c.logs = c.logs + "\n[DONE] " + fmt.Sprint(args...)
}

// Donef implements logger interface
func (c *CatchLogger) Donef(format string, args ...interface{}) {
	c.logs = c.logs + "\n[DONE] " + fmt.Sprintf(format, args...)
}

// Fail implements logger interface
func (c *CatchLogger) Fail(args ...interface{}) {
	c.logs = c.logs + "\n[FAIL] " + fmt.Sprint(args...)
}

// Failf implements logger interface
func (c *CatchLogger) Failf(format string, args ...interface{}) {
	c.logs = c.logs + "\n[FAIL] " + fmt.Sprintf(format, args...)
}

// Print implements logger interface
func (c *CatchLogger) Print(level logrus.Level, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		c.Info(args...)
	case logrus.DebugLevel:
		c.Debug(args...)
	case logrus.WarnLevel:
		c.Warn(args...)
	case logrus.ErrorLevel:
		c.Error(args...)
	case logrus.PanicLevel:
		c.Panic(args...)
	case logrus.FatalLevel:
		c.Fatal(args...)
	}
}

// Printf implements logger interface
func (c *CatchLogger) Printf(level logrus.Level, format string, args ...interface{}) {
	switch level {
	case logrus.InfoLevel:
		c.Infof(format, args...)
	case logrus.DebugLevel:
		c.Debugf(format, args...)
	case logrus.WarnLevel:
		c.Warnf(format, args...)
	case logrus.ErrorLevel:
		c.Errorf(format, args...)
	case logrus.PanicLevel:
		c.Panicf(format, args...)
	case logrus.FatalLevel:
		c.Fatalf(format, args...)
	}
}

// StartWait implements logger interface
func (c *CatchLogger) StartWait(message string) {
	c.logs = c.logs + "\n[WAIT] " + message
}

// StopWait implements logger interface
func (c *CatchLogger) StopWait() {
	c.logs = c.logs + "\n[STOPWAIT] "
}

// SetLevel implements logger interface
func (c *CatchLogger) SetLevel(level logrus.Level) {}

// GetLevel implements logger interface
func (c *CatchLogger) GetLevel() logrus.Level { return logrus.FatalLevel }

// Write implements logger interface
func (c *CatchLogger) Write(message []byte) (int, error) {
	c.logs = c.logs + string(message)
	return len(message), nil
}

// WriteString implements logger interface
func (c *CatchLogger) WriteString(message string) {
	c.logs = c.logs + message
}

// Question asks a new question
func (c *CatchLogger) Question(params *survey.QuestionOptions) (string, error) {
	return c.Survey.Question(params)
}
