package build

import (
	"context"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	ginkgo "github.com/loft-sh/devspace/e2e/ginkgo-ext"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/spf13/cobra"

	dockertypes "github.com/docker/docker/api/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("build", func() {
	var (
		f          *utils.BaseCustomFactory
		testDir    string
		tmpDir     string
		authConfig *dockertypes.AuthConfig
	)

	ginkgo.BeforeAll(func() {
		var err error
		testDir = "tests/build/testdata"

		tmpDir, _, err = utils.CreateTempDir()
		utils.ExpectNoError(err, "error creating tmp dir")

		// Copy the testdata into the temp dir
		err = utils.Copy(testDir, tmpDir)
		utils.ExpectNoError(err, "error copying test dir")

		f = utils.DefaultFactory

		dockerClient, err := f.NewDockerClient(f.GetLog())
		utils.ExpectNoError(err, "create docker client")
		authConfig, err = dockerClient.Login("hub.docker.com", "", "", true, false, false)
		if err != nil || authConfig.Username == "" {
			ginkgo.Skip("Can't login, skip kaniko " + err.Error())
		}

		// Create namespace
		_, err = f.Client.KubeClient().CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: f.Namespace,
			},
		}, metav1.CreateOptions{})
		if err != nil && err.Error() != "namespaces \"testns\" already exists" {
			utils.ExpectNoError(err, "error creating namespace")
		}
	})

	ginkgo.AfterAll(func() {
		utils.DeleteTempAndResetWorkingDir(tmpDir, f.Pwd, f.GetLog())
	})

	ginkgo.It("kaniko", func() {
		// Change working directory
		err := utils.ChangeWorkingDir(tmpDir+"/kaniko", f.GetLog())
		utils.ExpectNoError(err, "error changing directory")

		// Kaniko requires a dockerhub account
		buildCmd := &cmd.BuildCmd{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
				Vars:      []string{"DEVSPACE_USERNAME=" + authConfig.Username},
			},
			SkipPush: true,
		}

		err = buildCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
		utils.ExpectNoError(err, "executing command")
	})
})
