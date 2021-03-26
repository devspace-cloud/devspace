package print

import (
	"path/filepath"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	ginkgo "github.com/loft-sh/devspace/e2e/ginkgo-ext"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/log"
	fakelog "github.com/loft-sh/devspace/pkg/util/log/testing"
	"github.com/mgutz/ansi"

	"github.com/spf13/cobra"
)

var _ = ginkgo.Describe("dev", func() {
	var (
		f                   *utils.BaseCustomFactory
		testDir             string
		tmpDir              string
		logger              *fakelog.CatchLogger
		defaultLoggerBackup log.Logger
	)

	ginkgo.BeforeAll(func() {
		// Create tmp dir
		var err error
		testDir = "tests/print/testdata"
		tmpDir, _, err = utils.CreateTempDir()
		utils.ExpectNoError(err, "error creating tmp dir")

		// Copy the testdata into the temp dir
		err = utils.Copy(testDir, tmpDir)
		utils.ExpectNoError(err, "error copying test dir")

		// Make backup of logger
		defaultLoggerBackup = log.GetInstance()

		// Set factory
		f = utils.DefaultFactory
	})

	ginkgo.BeforeEach(func() {
		logger = fakelog.NewCatchLogger()
		log.SetInstance(logger)
		utils.DefaultFactory.CacheLogger = logger
	})

	ginkgo.AfterAll(func() {
		utils.DeleteTempAndResetWorkingDir(tmpDir, f.Pwd, f.GetLog())
		utils.DefaultFactory.CacheLogger = fakelog.NewFakeLogger()
		log.SetInstance(defaultLoggerBackup)
	})

	ginkgo.It("default", func() {
		// Change working directory
		err := utils.ChangeWorkingDir(filepath.Join(tmpDir, "default"), fakelog.NewFakeLogger())
		utils.ExpectNoError(err, "error changing directory")

		// Run cmd
		printCmd := &cmd.PrintCmd{
			GlobalFlags: &flags.GlobalFlags{
				Vars: []string{"MY_IMAGE=default"},
			},
		}
		err = printCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
		utils.ExpectNoError(err, "run cmd")

		// Check output
		logs := logger.GetLogs()
		utils.ExpectEqual(getExpectedForDefault(tmpDir), logs, "Wrong output")
	})
})

func getExpectedForDefault(tmpDir string) string {
	return `
-------------------

Vars:

` + ansi.Color(" Name  ", "green+b") + "    " + ansi.Color(" Value  ", "green+b") + "  " + `
 MY_IMAGE   default  


-------------------

Loaded path: ` + filepath.Join(tmpDir, "default", "devspace.yaml") + `

-------------------

version: v1beta9
images:
  default:
    image: dscr.io/user/devspaceprinttest
    preferSyncOverRebuild: true
deployments:
- name: devspace-print-test
  helm:
    componentChart: true
    values:
      containers:
      - env:
        - name: TEST_ENV
          value: development
        image: dscr.io/user/devspaceprinttest
      service:
        ports:
        - port: 8080
dev:
  ports:
  - imageName: default
    forward:
    - port: 8080
  open:
  - url: http://localhost:8080
  sync:
  - imageName: default
    excludePaths:
    - Dockerfile
    - devspace.yaml
`
}
