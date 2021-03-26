package render

import (
	"path/filepath"
	"regexp"

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
		f       *utils.BaseCustomFactory
		testDir string
		tmpDir  string
		logger  *fakelog.CatchLogger
	)

	ginkgo.BeforeAll(func() {
		// Create tmp dir
		var err error
		testDir = "tests/render/testdata"
		tmpDir, _, err = utils.CreateTempDir()
		utils.ExpectNoError(err, "error creating tmp dir")

		// Copy the testdata into the temp dir
		err = utils.Copy(testDir, tmpDir)
		utils.ExpectNoError(err, "error copying test dir")

		// Set factory
		f = utils.DefaultFactory
	})

	ginkgo.BeforeEach(func() {
		logger = fakelog.NewCatchLogger()
		utils.DefaultFactory.CacheLogger = logger
	})

	ginkgo.AfterAll(func() {
		utils.DeleteTempAndResetWorkingDir(tmpDir, f.Pwd, f.GetLog())
		utils.DefaultFactory.CacheLogger = fakelog.NewFakeLogger()
	})

	ginkgo.It("helm v2", func() {
		ginkgo.Skip("helm v2 makes trouble")
		runTest(f, logger, testCase{
			dir: filepath.Join(tmpDir, "helm_v2"),
			renderCmd: &cmd.RenderCmd{
				SkipPush:    true,
				GlobalFlags: &flags.GlobalFlags{},
			},
		})
	})

	ginkgo.It("helm v3", func() {
		runTest(f, logger, testCase{
			dir: filepath.Join(tmpDir, "helm_v3"),
			renderCmd: &cmd.RenderCmd{
				SkipPush:    true,
				GlobalFlags: &flags.GlobalFlags{},
				Writer:      logger,
			},
			expectedOutput: helmv3ExpectedOutput,
		})
	})

	ginkgo.It("kubectl", func() {
		runTest(f, logger, testCase{
			dir: filepath.Join(tmpDir, "kubectl"),
			renderCmd: &cmd.RenderCmd{
				SkipPush:    true,
				GlobalFlags: &flags.GlobalFlags{},
				Writer:      logger,
			},
			expectedOutput: kubectlExpectedOutput,
		})
	})
})

type testCase struct {
	dir            string
	renderCmd      *cmd.RenderCmd
	expectedOutput string
}

func runTest(f *utils.BaseCustomFactory, logger *fakelog.CatchLogger, testCase testCase) {
	// Change working directory
	err := utils.ChangeWorkingDir(testCase.dir, fakelog.NewFakeLogger())
	utils.ExpectNoError(err, "error changing directory")

	// Run cmd
	err = testCase.renderCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
	utils.ExpectNoError(err, "run cmd")

	// Check output
	logs := logger.GetLogs()
	match, err := regexp.MatchString(testCase.expectedOutput, logs)
	utils.ExpectNoError(err, "check with regex")
	utils.ExpectTrue(match, "Wrong output")
}

const helmv3ExpectedOutput = `
---
# Source: component-chart/templates/service\.yaml
apiVersion: v1
kind: Service
metadata:
  name: "quickstart"
  labels:
    "app\.kubernetes\.io/name": "quickstart"
    "app\.kubernetes\.io/managed-by": "Helm"
  annotations:
    "helm\.sh/chart": "component-chart-0\.7\.1"
spec:
  externalIPs:
  ports:
    - name: "port-0"
      port: 3000
      targetPort: 3000
      protocol: "TCP"
  selector:
    "app\.kubernetes\.io/name": "devspace-app"
    "app\.kubernetes\.io/component": "quickstart"
  type: "ClusterIP"
---
# Source: component-chart/templates/deployment\.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "quickstart"
  labels:
    "app\.kubernetes\.io/name": "devspace-app"
    "app\.kubernetes\.io/component": "quickstart"
    "app\.kubernetes\.io/managed-by": "Helm"
  annotations:
    "helm\.sh/chart": "component-chart-0\.7\.1"
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      "app\.kubernetes\.io/name": "devspace-app"
      "app\.kubernetes\.io/component": "quickstart"
      "app\.kubernetes\.io/managed-by": "Helm"
  template:
    metadata:
      labels:
        "app\.kubernetes\.io/name": "devspace-app"
        "app\.kubernetes\.io/component": "quickstart"
        "app\.kubernetes\.io/managed-by": "Helm"
      annotations:
        "helm\.sh/chart": "component-chart-0\.7\.1"
    spec:
      imagePullSecrets:
      nodeSelector:
        null
      nodeName:
        null
      affinity:
        null
      tolerations:
        null
      dnsConfig:
        null
      hostAliases:
        null
      overhead:
        null
      readinessGates:
        null
      securityContext:
        null
      topologySpreadConstraints:
        null
      terminationGracePeriodSeconds: 5
      ephemeralContainers:
        null
      containers:
        - image: "dscr\.io/rendertestuser/helmv3:[a-zA-Z]{7}"
          name: "container-0"
          command:
          args:
          env:
            null
          envFrom:
            null
          securityContext:
            null
          lifecycle:
            null
          livenessProbe:
            null
          readinessProbe:
            null
          startupProbe:
            null
          volumeDevices:
            null
          volumeMounts:
      initContainers:
      volumes:
  volumeClaimTemplates:
---
# Source: component-chart/templates/deployment\.yaml
# Create headless service for StatefulSet

`
const kubectlExpectedOutput = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: devspace
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app\.kubernetes\.io/component: default
      app\.kubernetes\.io/name: devspace-app
  template:
    metadata:
      labels:
        app\.kubernetes\.io/component: default
        app\.kubernetes\.io/name: devspace-app
    spec:
      containers:
      - image: dscr\.io/yourusername/quickstart:[a-zA-Z]{7}
        name: default

---
apiVersion: v1
kind: Service
metadata:
  labels:
    app\.kubernetes\.io/name: devspace-app
  name: external
  namespace: default
spec:
  ports:
  - name: port-0
    port: 80
    protocol: TCP
    targetPort: 3000
  selector:
    app\.kubernetes\.io/component: default
    app\.kubernetes.io/name: devspace-app
  type: ClusterIP

---
`
