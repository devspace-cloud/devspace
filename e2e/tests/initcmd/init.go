package initcmd

import (
	"github.com/loft-sh/devspace/cmd"
	ginkgo "github.com/loft-sh/devspace/e2e/ginkgo-ext"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/build/builder/helper"
	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/util/ptr"
	"github.com/spf13/cobra"
)

var _ = ginkgo.Describe("init", func() {
	var (
		f       *customFactory
		testDir string
		tmpDir  string
	)

	ginkgo.BeforeAll(func() {
		var err error
		testDir = "tests/initcmd/testdata"
		tmpDir, _, err = utils.CreateTempDir()
		utils.ExpectNoError(err, "error creating tmp dir")

		// Copy the testdata into the temp dir
		err = utils.Copy(testDir, tmpDir)
		utils.ExpectNoError(err, "error copying test dir")
		f = &customFactory{
			utils.DefaultFactory,
		}
	})

	ginkgo.AfterAll(func() {
		utils.DeleteTempAndResetWorkingDir(tmpDir, f.Pwd, f.GetLog())
	})

	ginkgo.It("Create dockerfile", func() {
		runTest(f, initTestCase{
			dir:     tmpDir + "/data1",
			answers: []string{cmd.CreateDockerfileOption, "go", "Use hub.docker.com => you are logged in as user", "user/data1", "build", "8080", cmd.ComponentChartOption},
			expected: config.NewConfig(map[interface{}]interface{}{
				"profiles": []interface{}{
					map[interface{}]interface{}{
						"name": "production",
						"patches": []interface{}{
							map[interface{}]interface{}{
								"op":   "remove",
								"path": "images.app.appendDockerfileInstructions",
							},
							map[interface{}]interface{}{
								"op":   "remove",
								"path": "images.app.injectRestartHelper",
							},
							map[interface{}]interface{}{
								"op":   "remove",
								"path": "images.app.rebuildStrategy",
							},
							map[interface{}]interface{}{
								"op":   "remove",
								"path": "images.app.build.docker.options.target",
							},
						},
					},
					map[interface{}]interface{}{
						"patches": []interface{}{
							map[interface{}]interface{}{
								"path": "dev.interactive",
								"value": map[interface{}]interface{}{
									"defaultEnabled": true,
								},
								"op": "add",
							},
							map[interface{}]interface{}{
								"op":    "add",
								"path":  "images.app.entrypoint",
								"value": []interface{}{"sleep", "9999999999"},
							},
						},
						"name": "interactive",
					},
				},
				"version": "v1beta9",
				"images": map[interface{}]interface{}{
					"app": map[interface{}]interface{}{
						"image":                        "user/data1",
						"injectRestartHelper":          true,
						"appendDockerfileInstructions": []interface{}{"USER root"},
						"rebuildStrategy":              "ignoreContextChanges",
						"build": map[interface{}]interface{}{
							"docker": map[interface{}]interface{}{
								"options": map[interface{}]interface{}{
									"target": "build",
								},
							},
						},
					},
				},
				"deployments": []interface{}{
					map[interface{}]interface{}{
						"name": "data1",
						"helm": map[interface{}]interface{}{
							"componentChart": true,
							"values": map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "user/data1",
									},
								},
								"service": map[interface{}]interface{}{
									"ports": []interface{}{
										map[interface{}]interface{}{
											"port": 8080,
										},
									},
								},
							},
						},
					},
				},
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"forward": []interface{}{
								map[interface{}]interface{}{
									"port": 8080,
								},
							},
							"imageName": "app",
						},
					},
					"open": []interface{}{
						map[interface{}]interface{}{
							"url": "http://localhost:8080",
						},
					},
					"sync": []interface{}{
						map[interface{}]interface{}{
							"imageName":          "app",
							"excludePaths":       []interface{}{".git/"},
							"uploadExcludePaths": []interface{}{"Dockerfile", "devspace.yaml"},
							"onUpload": map[interface{}]interface{}{
								"restartContainer": true,
							},
						},
					},
				},
			}, &latest.Config{
				Version: latest.Version,
				Images: map[string]*latest.ImageConfig{
					"app": &latest.ImageConfig{
						Image:                        "user/data1",
						PreferSyncOverRebuild:        false,
						InjectRestartHelper:          true,
						AppendDockerfileInstructions: []string{"USER root"},
						RebuildStrategy:              "ignoreContextChanges",
						Build: &latest.BuildConfig{
							Docker: &latest.DockerConfig{
								Options: &latest.BuildOptions{
									Target: "build",
								},
							},
						},
					},
				},
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "data1",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "user/data1",
									},
								},
								"service": map[interface{}]interface{}{
									"ports": []interface{}{
										map[interface{}]interface{}{
											"port": 8080,
										},
									},
								},
							},
						},
					},
				},
				Dev: &latest.DevConfig{
					Ports: []*latest.PortForwardingConfig{
						&latest.PortForwardingConfig{
							ImageName: "app",
							PortMappings: []*latest.PortMapping{
								&latest.PortMapping{
									LocalPort: ptr.Int(8080),
								},
							},
						},
					},
					Open: []*latest.OpenConfig{
						&latest.OpenConfig{
							URL: "http://localhost:8080",
						},
					},
					Sync: []*latest.SyncConfig{
						&latest.SyncConfig{
							ImageName:          "app",
							ExcludePaths:       []string{".git/"},
							UploadExcludePaths: []string{"Dockerfile", "devspace.yaml"},
							OnUpload: &latest.SyncOnUpload{
								RestartContainer: true,
							},
						},
					},
				},
			}, &generated.Config{
				Vars:     map[string]string{},
				Profiles: map[string]*generated.CacheConfig{},
			}, map[string]interface{}{}),
		})
	})

	ginkgo.It("Everything already created", func() {
		runTest(f, initTestCase{
			dir: tmpDir + "/everythingIsThere",
			expected: config.NewConfig(map[interface{}]interface{}{
				"profiles": []interface{}{
					map[interface{}]interface{}{
						"name": "production",
						"patches": []interface{}{
							map[interface{}]interface{}{
								"op":   "remove",
								"path": "images.app.appendDockerfileInstructions",
							},
							map[interface{}]interface{}{
								"op":   "remove",
								"path": "images.app.injectRestartHelper",
							},
						},
					},
					map[interface{}]interface{}{
						"patches": []interface{}{
							map[interface{}]interface{}{
								"path": "dev.interactive",
								"value": map[interface{}]interface{}{
									"defaultEnabled": true,
								},
								"op": "add",
							},
							map[interface{}]interface{}{
								"op":    "add",
								"path":  "images.app.entrypoint",
								"value": []interface{}{"sleep", "9999999999"},
							},
						},
						"name": "interactive",
					},
				},
				"version": "v1beta9",
				"images": map[interface{}]interface{}{
					"app": map[interface{}]interface{}{
						"image":                        "user/everythingIsThere",
						"injectRestartHelper":          true,
						"appendDockerfileInstructions": []interface{}{"USER root"},
						"preferSyncOverRebuild":        true,
					},
				},
				"deployments": []interface{}{
					map[interface{}]interface{}{
						"name": "everythingIsThere",
						"helm": map[interface{}]interface{}{
							"componentChart": true,
							"values": map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "user/everythingIsThere",
									},
								},
								"service": map[interface{}]interface{}{
									"ports": []interface{}{
										map[interface{}]interface{}{
											"port": 8081,
										},
									},
								},
							},
						},
					},
				},
				"dev": map[interface{}]interface{}{
					"ports": []interface{}{
						map[interface{}]interface{}{
							"forward": []interface{}{
								map[interface{}]interface{}{
									"port": 8081,
								},
							},
							"imageName": "app",
						},
					},
					"open": []interface{}{
						map[interface{}]interface{}{
							"url": "http://localhost:8081",
						},
					},
					"sync": []interface{}{
						map[interface{}]interface{}{
							"imageName":          "app",
							"excludePaths":       []interface{}{".git/"},
							"uploadExcludePaths": []interface{}{"Dockerfile", "devspace.yaml"},
							"onUpload": map[interface{}]interface{}{
								"restartContainer": true,
							},
						},
					},
				},
			}, &latest.Config{
				Version: latest.Version,
				Images: map[string]*latest.ImageConfig{
					"app": &latest.ImageConfig{
						Image:                        "user/everythingIsThere",
						PreferSyncOverRebuild:        true,
						InjectRestartHelper:          true,
						AppendDockerfileInstructions: []string{"USER root"},
					},
				},
				Deployments: []*latest.DeploymentConfig{
					&latest.DeploymentConfig{
						Name: "everythingIsThere",
						Helm: &latest.HelmConfig{
							ComponentChart: ptr.Bool(true),
							Values: map[interface{}]interface{}{
								"containers": []interface{}{
									map[interface{}]interface{}{
										"image": "user/everythingIsThere",
									},
								},
								"service": map[interface{}]interface{}{
									"ports": []interface{}{
										map[interface{}]interface{}{
											"port": 8081,
										},
									},
								},
							},
						},
					},
				},
				Dev: &latest.DevConfig{
					Ports: []*latest.PortForwardingConfig{
						&latest.PortForwardingConfig{
							ImageName: "app",
							PortMappings: []*latest.PortMapping{
								&latest.PortMapping{
									LocalPort: ptr.Int(8081),
								},
							},
						},
					},
					Open: []*latest.OpenConfig{
						&latest.OpenConfig{
							URL: "http://localhost:8081",
						},
					},
					Sync: []*latest.SyncConfig{
						&latest.SyncConfig{
							ImageName:          "app",
							ExcludePaths:       []string{".git/"},
							UploadExcludePaths: []string{"Dockerfile", "devspace.yaml"},
							OnUpload: &latest.SyncOnUpload{
								RestartContainer: true,
							},
						},
					},
				},
			}, &generated.Config{
				Vars:     map[string]string{},
				Profiles: map[string]*generated.CacheConfig{},
			}, map[string]interface{}{}),
		})
	})
})

type initTestCase struct {
	dir     string
	answers []string

	expected config.Config
}

func runTest(f *customFactory, testCase initTestCase) {
	err := utils.ChangeWorkingDir(testCase.dir, f.GetLog())
	utils.ExpectNoError(err, "error changing directory")

	initCmd := cmd.InitCmd{
		Dockerfile:  helper.DefaultDockerfilePath,
		Reconfigure: false,
		Context:     "",
		Provider:    "",
	}

	survey, err := f.GetSurvey()
	utils.ExpectNoError(err, "get survey")
	for _, answer := range testCase.answers {
		survey.SetNextAnswer(answer)
	}

	err = initCmd.Run(f, []plugin.Metadata{}, &cobra.Command{}, []string{})
	utils.ExpectNoError(err, "executing command")

	if testCase.expected != nil {
		config, err := f.NewConfigLoader("").Load(nil, f.GetLog())
		utils.ExpectNoError(err, "new config loader")

		utils.ExpectEqual(config, testCase.expected, "Unexpected config")
	}
}
