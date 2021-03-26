package dev

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/loft-sh/devspace/cmd"
	"github.com/loft-sh/devspace/cmd/flags"
	ginkgo "github.com/loft-sh/devspace/e2e/ginkgo-ext"
	"github.com/loft-sh/devspace/e2e/tests/deploy"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/fsutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("dev", func() {
	var (
		f       *customFactory
		testDir string
		tmpDir  string
	)

	ginkgo.BeforeAll(func() {
		var err error
		testDir = "tests/dev/testdata"
		tmpDir, _, err = utils.CreateTempDir()
		utils.ExpectNoError(err, "error creating tmp dir")

		// Copy the testdata into the temp dir
		err = utils.Copy(testDir, tmpDir)
		utils.ExpectNoError(err, "error copying test dir")

		f = &customFactory{
			BaseCustomFactory: utils.DefaultFactory,
		}

		if runtime.GOOS == "linux" {
			// Change working directory
			err := utils.ChangeWorkingDir(tmpDir, f.GetLog())
			utils.ExpectNoError(err, "error changing directory")

			err = exec.Command("chmod", "u+x", "custom-builder/custom/build").Start()
			utils.ExpectNoError(err, "enabling custom build")
		}
	})

	ginkgo.BeforeEach(func() {
		f.interruptSync = make(chan error)
		f.interruptPortforward = make(chan error)
		f.enableSync = make(chan bool)
	})

	ginkgo.AfterEach(func() {
		close(f.interruptPortforward)
		close(f.interruptSync)
		for _, deployment := range []string{"php-app", "custom-builder-deployment", "root", "dependency1", "dependency2", "dependency3", "dependency4", "remote-dependency", "hook-deployment"} {
			f.Client.KubeClient().AppsV1().Deployments(f.Namespace).Delete(context.Background(), deployment, metav1.DeleteOptions{})
		}
	})

	ginkgo.AfterAll(func() {
		utils.DeleteTempAndResetWorkingDir(tmpDir, f.Pwd, f.GetLog())
	})

	ginkgo.It("custom-builder", func() {
		err := runTest(f, devTestCase{
			dir:          path.Join(tmpDir, "custom-builder"),
			deployments:  []string{"custom-builder-deployment"},
			portAndPaths: map[int]string{},
			devCmd: &cmd.DevCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.Namespace,
				},
				Terminal:       true,
				Sync:           true,
				Portforwarding: true,
				ForceBuild:     true,
				SkipPush:       true,
			},
		})
		utils.ExpectNoError(err, "Test fail")
	})

	ginkgo.It("default", func() {
		err := runTest(f, devTestCase{
			dir:         path.Join(tmpDir, "default"),
			deployments: []string{"php-app"},
			portAndPaths: map[int]string{
				8080: "/index.php",
				1234: "",
			},
			devCmd: &cmd.DevCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.Namespace,
				},
				Terminal:       true,
				UI:             true,
				UIPort:         1234,
				Sync:           true,
				Portforwarding: true,
				ForceBuild:     true,
				SkipPush:       true,
			},
		})
		utils.ExpectNoError(err, "Test fail")

		// Check sync
		checkSync(f, path.Join(tmpDir, "default", "foo"), "/syncInThis")

		// Delete Running pods
		pods, err := f.BaseCustomFactory.Client.KubeClient().CoreV1().Pods(f.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: "release=php-app",
		})
		utils.ExpectNoError(err, "get pods")
		for _, pod := range pods.Items {
			f.BaseCustomFactory.Client.KubeClient().CoreV1().Pods(f.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
		}

		// Wait for restart
		wait.PollImmediate(time.Second, time.Second*15, func() (bool, error) {
			pods, err := f.BaseCustomFactory.Client.KubeClient().CoreV1().Pods(f.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: "release=php-app",
			})
			if err != nil {
				err = errors.Errorf("get pods: %v", err)
				return false, nil
			}
			if len(pods.Items) == 0 {
				err = errors.Errorf("No pods")
				return false, nil
			}
			for _, pod := range pods.Items {
				if pod.Status.Reason != "Running" {
					err = errors.Errorf("Pod %s in Status %s", pod.Name, pod.Status.Reason)
					return false, nil
				}
			}
			return true, nil
		})
		utils.ExpectNoError(err, "wait for pods to restart")

		// Check sync after restart
		checkResync(f, path.Join(tmpDir, "default", "foo"), "/syncInThis")
	})

	ginkgo.It("dependencies", func() {
		err := runTest(f, devTestCase{
			dir:         path.Join(tmpDir, "dependencies"),
			deployments: []string{"root", "dependency3", "dependency4"},
			devCmd: &cmd.DevCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.Namespace,
				},
				Sync:           true,
				Portforwarding: false,
				ForceBuild:     true,
				SkipPush:       true,
			},
		})
		utils.ExpectNoError(err, "Test fail")
	})

	ginkgo.It("hooks", func() {
		if runtime.GOOS == "windows" {
			ginkgo.Skip("Skip on windows")
		}

		err := runTest(f, devTestCase{
			dir:         path.Join(tmpDir, "hooks"),
			deployments: []string{"hook-deployment"},
		})
		utils.ExpectNoError(err, "Test fail")
	})
})

type devTestCase struct {
	dir          string
	deployments  []string
	portAndPaths map[int]string
	devCmd       *cmd.DevCmd
}

func runTest(f *customFactory, testCase devTestCase) error {
	// Change working directory
	err := utils.ChangeWorkingDir(testCase.dir, f.GetLog())
	utils.ExpectNoError(err, "error changing directory")

	// Exec command
	errChan := make(chan error)
	syncErrChan := sync.Mutex{}
	go func() {
		if testCase.devCmd == nil {
			testCase.devCmd = &cmd.DevCmd{
				GlobalFlags: &flags.GlobalFlags{
					Namespace: f.Namespace,
				},
				Terminal:       true,
				Sync:           true,
				Portforwarding: true,
				ForceBuild:     true,
				SkipPush:       true,
			}
		}
		err = testCase.devCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})

		syncErrChan.Lock()
		if errChan == nil {
			utils.ExpectNoError(err, "dev command returned an error")
		} else {
			go func() {
				errChan <- err
			}()
		}
		syncErrChan.Unlock()
	}()

	defer func() {
		// Check if dev-cmd returned an error
		syncErrChan.Lock()
		select {
		case err = <-errChan:
			utils.ExpectNoError(err, "dev command returned an error")
		default:
			close(errChan)
			errChan = nil
		}
		syncErrChan.Unlock()
	}()

	// Check deployments
	err = deploy.CheckDeployments(f.BaseCustomFactory, testCase.deployments)
	if err != nil {
		return err
	}

	// Check Port forwarding
	for port, path := range testCase.portAndPaths {
		err = checkPortWorking(port, path)
		if err != nil && port != 8080 {
			utils.ExpectNoError(err, "port forwarding doesn't work for port "+fmt.Sprint(port))
		}
	}

	return nil
}

func printDeployments(f *customFactory) {
	deployments, _ := f.Client.KubeClient().AppsV1().Deployments(f.Namespace).List(context.Background(), metav1.ListOptions{})
	asJSON, _ := json.Marshal(deployments.Items)
	ginkgo.Skip(string(asJSON))
}

func checkSync(f *customFactory, localPath, containerPath string) {
	var err error

	// Create local files
	fsutil.WriteToFile([]byte("321"), path.Join("foo", "upstreamThis"))
	fsutil.WriteToFile([]byte("321"), path.Join("foo", "excludeThis2"))

	// Start initial sync
	f.enableSync <- true

	// Check inital sync
	enterCmd := cmd.EnterCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
		},
	}
	wait.PollImmediate(time.Second, time.Second*15, func() (bool, error) {
		// Check if upstream works
		err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "cat " + path.Join(containerPath, "upstreamThis")})
		if err != nil {
			err = errors.Errorf("get upstreamed file %v", err)
			return false, nil
		}
		return true, nil
	})
	utils.ExpectNoError(err, "check initial sync")

	// Create files in container
	err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "echo 123 > " + path.Join(containerPath, "downstreamThis")})
	utils.ExpectNoError(err, "executing enter command")
	err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "echo 123 > " + path.Join(containerPath, "excludeThis")})
	utils.ExpectNoError(err, "executing enter command")

	wait.PollImmediate(time.Second, time.Second*15, func() (bool, error) {
		var content []byte

		// Check if downstream works
		content, err = fsutil.ReadFile(path.Join(localPath, "downstreamThis"), -1)
		if err != nil {
			err = errors.Errorf("get downstreamed file %v", err)
			return false, nil
		}
		if string(content) != "123\n" {
			err = errors.Errorf("Unexpected content of file downstreamThis. \nExpected: %s\nActual: %s", "123\n", string(content))
			return false, nil
		}

		// Check if file is still there
		err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "cat " + path.Join(containerPath, "upstreamThis")})
		if err != nil {
			err = errors.Errorf("get upstreamed file %v", err)
			return false, nil
		}

		// Check if exclusions work
		_, err = fsutil.ReadFile(path.Join(localPath, "excludeThis"), -1)
		if err == nil {
			err = errors.Errorf("excluded file was downstreamed")
			return true, nil
		}
		err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "cat " + path.Join(containerPath, "excludeThis2")})
		if err == nil {
			err = errors.Errorf("excluded file was upstreamed")
			return true, nil
		}

		err = nil
		return true, nil
	})
	utils.ExpectNoError(err, "check sync")
}

func checkResync(f *customFactory, localPath, containerPath string) {
	// Create files in container
	enterCmd := cmd.EnterCmd{
		GlobalFlags: &flags.GlobalFlags{
			Namespace: f.Namespace,
		},
	}
	err := enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "echo 1234 > " + path.Join(containerPath, "downstreamThis2")})
	utils.ExpectNoError(err, "executing enter command")

	// Create local files
	fsutil.WriteToFile([]byte("4321"), path.Join(localPath, "upstreamThis2"))

	wait.PollImmediate(time.Second, time.Second*25, func() (bool, error) {
		var content []byte

		// Check if downstream works
		content, err = fsutil.ReadFile(path.Join(localPath, "downstreamThis"), -1)
		if err != nil {
			err = errors.Errorf("get downstreamed file %v", err)
			return false, nil
		}
		if string(content) != "123\n" {
			err = errors.Errorf("Unexpected content of file downstreamThis. \nExpected: %s\nActual: %s", "123\n", string(content))
			return false, nil
		}
		content, err = fsutil.ReadFile(path.Join(localPath, "downstreamThis2"), -1)
		if err != nil {
			err = errors.Errorf("get downstreamed file %v", err)
			return false, nil
		}
		if string(content) != "1234\n" {
			err = errors.Errorf("Unexpected content of file downstreamThis2. \nExpected: %s\nActual: %s", "1234\n", string(content))
			return false, nil
		}

		// Check if upstream works
		err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "cat " + path.Join(containerPath, "downstreamThis")})
		if err != nil {
			err = errors.Errorf("get upstreamed file %v", err)
			return false, nil
		}
		err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "cat " + path.Join(containerPath, "upstreamThis")})
		if err != nil {
			err = errors.Errorf("get upstreamed file %v", err)
			return false, nil
		}
		err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "cat " + path.Join(containerPath, "upstreamThis2")})
		if err != nil {
			err = errors.Errorf("get upstreamed file %v", err)
			return false, nil
		}

		// Check if exclusions work
		_, err = fsutil.ReadFile(path.Join(localPath, "excludeThis"), -1)
		if err == nil {
			err = errors.Errorf("excluded file was downstreamed")
			return true, nil
		}
		err = enterCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{"sh", "-c", "cat " + path.Join(containerPath, "excludeThis2")})
		if err == nil {
			err = errors.Errorf("excluded file was upstreamed")
			return true, nil
		}

		err = nil
		return true, nil
	})
	utils.ExpectNoError(err, "check sync")
}

func checkPortWorking(port int, path string) error {
	var err error
	wait.PollImmediate(time.Second, time.Second*15, func() (bool, error) {
		_, err = http.Get(fmt.Sprintf("http://localhost:%d%s", port, path))
		return err == nil, nil
	})
	return err
}
