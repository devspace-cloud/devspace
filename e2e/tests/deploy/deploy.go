package deploy

import (
	"context"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	ginkgo "github.com/loft-sh/devspace/e2e/ginkgo-ext"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var _ = ginkgo.Describe("deploy", func() {
	var (
		f       *utils.BaseCustomFactory
		testDir string
		tmpDir  string
	)

	ginkgo.BeforeAll(func() {
		var err error
		testDir = "tests/deploy/testdata"

		tmpDir, _, err = utils.CreateTempDir()
		utils.ExpectNoError(err, "error creating tmp dir")

		// Copy the testdata into the temp dir
		err = utils.Copy(testDir, tmpDir)
		utils.ExpectNoError(err, "error copying test dir")

		f = utils.DefaultFactory
	})

	ginkgo.AfterEach(func() {
		for _, deployment := range []string{"dependency1", "dependency2", "php-app", "remote-dependency", "root", "tiller-deploy", "helmv3deploy", "helmv2deploy", "kubedeploy", "kustomize-deploy"} {
			f.Client.KubeClient().AppsV1().Deployments(f.Namespace).Delete(context.Background(), deployment, metav1.DeleteOptions{})
		}
	})

	ginkgo.AfterAll(func() {
		utils.DeleteTempAndResetWorkingDir(tmpDir, f.Pwd, f.GetLog())
	})

	ginkgo.It("Component chart", func() {
		// Change working directory
		err := utils.ChangeWorkingDir(tmpDir+"/default", f.GetLog())
		utils.ExpectNoError(err, "error changing directory")

		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
				Silent:    true,
			},
			SkipPush: true,
		}

		err = deployCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
		utils.ExpectNoError(err, "executing command")

		err = CheckDeployments(f, []string{"dependency1", "dependency2", "php-app", "remote-dependency", "root", "tiller-deploy"})
	})

	ginkgo.It("Helm v3 with kaniko", func() {
		ginkgo.Skip("Kaniko doesn't work in a CI pipeline")

		// Change working directory
		err := utils.ChangeWorkingDir(tmpDir+"/helm", f.GetLog())
		utils.ExpectNoError(err, "error changing directory")

		dockerClient, err := f.NewDockerClient(f.GetLog())
		utils.ExpectNoError(err, "create docker client")
		authConfig, err := dockerClient.Login("hub.docker.com", "", "", true, false, false)
		if err != nil || authConfig.Username == "" {
			ginkgo.Skip("Can't login, skip kaniko " + err.Error())
		}

		// Kaniko requires a dockerhub account
		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
				Vars:      []string{"DEVSPACE_USERNAME=" + authConfig.Username},
			},
		}

		err = deployCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
		utils.ExpectNoError(err, "executing command")

		err = CheckDeployments(f, []string{"helmv3deploy"})
		utils.ExpectNoError(err, "check deployments")
	})

	ginkgo.It("Helm v2", func() {
		ginkgo.Skip("Helm v2 is troublesome")
		// Change working directory
		err := utils.ChangeWorkingDir(tmpDir+"/helm_v2", f.GetLog())
		utils.ExpectNoError(err, "error changing directory")

		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
			},
		}

		err = deployCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
		utils.ExpectNoError(err, "executing command")

		err = CheckDeployments(f, []string{"helmv2deploy", "tiller-deploy"})
		utils.ExpectNoError(err, "check deployments")
	})

	ginkgo.It("Kube yamls", func() {
		// Change working directory
		err := utils.ChangeWorkingDir(tmpDir+"/kubectl", f.GetLog())
		utils.ExpectNoError(err, "error changing directory")

		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
			},
			SkipPush: true,
		}

		err = deployCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
		utils.ExpectNoError(err, "executing command")

		err = CheckDeployments(f, []string{"kubedeploy"})
		utils.ExpectNoError(err, "check deployments")
	})

	ginkgo.It("Kustomize", func() {
		// Change working directory
		err := utils.ChangeWorkingDir(tmpDir+"/kustomize", f.GetLog())
		utils.ExpectNoError(err, "error changing directory")

		deployCmd := &cmd.DeployCmd{
			GlobalFlags: &flags.GlobalFlags{
				Namespace: f.Namespace,
				NoWarn:    true,
			},
			SkipPush: true,
		}

		err = deployCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
		utils.ExpectNoError(err, "executing command")

		err = CheckDeployments(f, []string{"kustomize-deploy"})
		utils.ExpectNoError(err, "check deployments")
	})
})

// CheckDeployments is a helper function that checks whether all listed deployments start including all their replicas
func CheckDeployments(f *utils.BaseCustomFactory, deployments []string) error {
	var err error
	wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		for _, deployment := range deployments {
			err = checkDeployment(f.Client.KubeClient(), f.Namespace, deployment)
			if err != nil {
				err = errors.Errorf("check deployment %s: %v", deployment, err)
				return false, nil
			}
		}
		return true, nil
	})
	return err
}

func checkDeployment(client kubernetes.Interface, namespace, name string) error {
	deployment, err := client.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if *deployment.Spec.Replicas == 0 {
		return errors.New("no replicas")
	}
	if *deployment.Spec.Replicas != deployment.Status.ReadyReplicas {
		return errors.New("replicas not ready")
	}
	return nil
}
