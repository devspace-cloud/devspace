package e2e

import (
	"fmt"
	"os"
	"testing"

	ginkgo "github.com/loft-sh/devspace/e2e/ginkgo-ext"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/util/log"
	fakelog "github.com/loft-sh/devspace/pkg/util/log/testing"
	"github.com/onsi/gomega"

	// Register tests
	_ "github.com/loft-sh/devspace/e2e/tests/build"
	_ "github.com/loft-sh/devspace/e2e/tests/deploy"
	_ "github.com/loft-sh/devspace/e2e/tests/dev"
	_ "github.com/loft-sh/devspace/e2e/tests/enter"
	_ "github.com/loft-sh/devspace/e2e/tests/initcmd"
	_ "github.com/loft-sh/devspace/e2e/tests/print"
	_ "github.com/loft-sh/devspace/e2e/tests/render"
)

var ()

func setupFactory() {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	utils.DefaultFactory.Namespace = "testns"
	utils.DefaultFactory.Verbose = false
	utils.DefaultFactory.Pwd = pwd
	utils.DefaultFactory.CacheLogger = fakelog.NewFakeLogger()
	utils.DefaultFactory.Client, err = utils.DefaultFactory.NewKubeClientFromContext("", utils.DefaultFactory.Namespace, false)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log.SetFakeFileLogger("default", fakelog.NewFakeLogger())
	log.SetFakeFileLogger("errors", fakelog.NewFakeLogger())
	log.SetFakeFileLogger("portforwarding", fakelog.NewFakeLogger())
	log.SetFakeFileLogger("sync", fakelog.NewFakeLogger())
}

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func RunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	ginkgo.RunSpecs(t, "Kubernetes e2e suite")
}
