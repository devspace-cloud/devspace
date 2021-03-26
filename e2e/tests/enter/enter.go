package enter

import (
	"os"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	ginkgo "github.com/loft-sh/devspace/e2e/ginkgo-ext"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	fakelog "github.com/loft-sh/devspace/pkg/util/log/testing"
	"github.com/spf13/cobra"
)

var _ = ginkgo.Describe("dev", func() {
	var (
		f            *utils.BaseCustomFactory
		testDir      string
		tmpDir       string
		stdOutBackup *os.File
		stdoutReader *os.File
	)

	ginkgo.BeforeAll(func() {
		// Create tmp dir
		var err error
		testDir = "tests/enter/testdata"
		tmpDir, _, err = utils.CreateTempDir()
		utils.ExpectNoError(err, "error creating tmp dir")

		// Copy the testdata into the temp dir
		err = utils.Copy(testDir, tmpDir)
		utils.ExpectNoError(err, "error copying test dir")

		// Make backup of stdout
		stdOutBackup = os.Stdout

		// Set factory
		f = utils.DefaultFactory
	})

	ginkgo.BeforeEach(func() {
		r, w, err := os.Pipe()
		utils.ExpectNoError(err, "create io pipe for stdout")
		os.Stdout = w
		stdoutReader = r
	})

	ginkgo.AfterAll(func() {
		os.Stdout = stdOutBackup
		utils.DeleteTempAndResetWorkingDir(tmpDir, f.Pwd, f.GetLog())
	})

	ginkgo.It("default", func() {

		// Change working directory
		err := utils.ChangeWorkingDir(tmpDir, fakelog.NewFakeLogger())
		utils.ExpectNoError(err, "error changing directory")

		// run dev command
		devCmd := &cmd.DevCmd{
			Wait:            true,
			ExitAfterDeploy: true,
			GlobalFlags: &flags.GlobalFlags{
				Silent: true,
			},
		}
		err = devCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
		utils.ExpectNoError(err, "run dev command")

		// run enter command
		enterCmd := &cmd.EnterCmd{
			GlobalFlags: &flags.GlobalFlags{
				Silent: true,
			},
		}
		err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"echo", "enter test hello"})
		utils.ExpectNoError(err, "run enter command")

		// check output
		buf := make([]byte, 1024)
		n, err := stdoutReader.Read(buf)
		utils.ExpectNoError(err, "read from output")
		utils.ExpectEqual(string(buf[:n]), "enter test hello\n", "Unexpected output")
	})
})
