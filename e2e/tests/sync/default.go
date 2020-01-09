package sync

import (
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/cmd"
	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/e2e/utils"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/pkg/errors"
)

func runDefault(f *customFactory, logger log.Logger) error {
	logger.Info("Run test 'default' of 'sync'")
	logger.StartWait("Run test...")
	defer logger.StopWait()

	sc := &cmd.SyncCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
			Silent:    true,
		},
		LocalPath:     "./../bla",
		ContainerPath: "/app",
		NoWatch:       true,
	}

	ec := &cmd.EnterCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
			NoWarn:    true,
			Silent:    true,
		},
		Container: "container-0",
	}

	err := sc.Run(f, nil, nil)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	done := utils.Capture()

	err = ec.Run(f, nil, []string{"ls", "foo"})
	if err != nil {
		return err
	}

	capturedOutput, err := done()
	if err != nil {
		return err
	}

	capturedOutput = strings.TrimSpace(capturedOutput)

	if strings.Index(capturedOutput, "foo.go") == -1 {
		return errors.Errorf("capturedOutput '%v' is different than output 'foo.go' for the enter cmd", capturedOutput)
	}

	return nil
}
