package build

import (
	"errors"
	"reflect"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/build/builder/custom"
	dockerbuilder "github.com/devspace-cloud/devspace/pkg/devspace/build/builder/docker"
	"github.com/devspace-cloud/devspace/pkg/devspace/build/builder/kaniko"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	fakedocker "github.com/devspace-cloud/devspace/pkg/devspace/docker/testing"
	fakekube "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	dockertypes "github.com/docker/docker/api/types"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"k8s.io/client-go/kubernetes/fake"
)

type createBuilderTestCase struct {
	name string

	imageConfigName string
	imageConf       *latest.ImageConfig
	imageTag        string
	options         Options
	pingErr         error

	expectedErr     string
	expectedBuilder interface{}
}

func TestCreateBuilder(t *testing.T) {
	fakeDocker := &fakedocker.FakeClient{
		AuthConfig: &dockertypes.AuthConfig{},
	}
	fakeClient := &fakekube.Client{
		Client: fake.NewSimpleClientset(),
	}

	kanikoBuilder, _ := kaniko.NewBuilder(nil, fakeDocker, fakeClient, "imageConfigName2", &latest.ImageConfig{
		Image: "imagename2",
		Build: &latest.BuildConfig{
			Kaniko: &latest.KanikoConfig{},
		},
	}, "imagetag2", false, log.Discard)
	dockerBuilder, _ := dockerbuilder.NewBuilder(nil, fakeDocker, fakeClient, "imageConfigName2", &latest.ImageConfig{
		Image: "imagename2",
		Build: &latest.BuildConfig{
			Kaniko: &latest.KanikoConfig{},
		},
	}, "imagetag2", true, false)

	testCases := []createBuilderTestCase{
		createBuilderTestCase{
			name:            "Create custom builder",
			imageConfigName: "imageConfigName",
			imageConf: &latest.ImageConfig{
				Build: &latest.BuildConfig{
					Custom: &latest.CustomConfig{},
				},
			},
			imageTag: "imageTag",
			expectedBuilder: custom.NewBuilder("imageConfigName", &latest.ImageConfig{
				Build: &latest.BuildConfig{
					Custom: &latest.CustomConfig{},
				},
			}, "imageTag"),
		},
		createBuilderTestCase{
			name:            "Create kaniko builder",
			imageConfigName: "imageConfigName2",
			imageConf: &latest.ImageConfig{
				Image: "imagename2",
				Build: &latest.BuildConfig{
					Kaniko: &latest.KanikoConfig{},
				},
			},
			imageTag:        "imagetag2",
			expectedBuilder: kanikoBuilder,
		},
		createBuilderTestCase{
			name:            "Create docker builder",
			imageConfigName: "imageConfigName3",
			imageConf: &latest.ImageConfig{
				Image: "imagename3",
				Build: &latest.BuildConfig{
					Docker: &latest.DockerConfig{
						PreferMinikube: ptr.Bool(false),
					},
				},
			},
			imageTag:        "imagetag3",
			expectedBuilder: dockerBuilder,
		},
		createBuilderTestCase{
			name:            "Fallback from docker to kaniko",
			imageConfigName: "imageConfigName2",
			imageConf: &latest.ImageConfig{
				Image: "imagename2",
				Build: &latest.BuildConfig{
					Docker: &latest.DockerConfig{},
				},
			},
			pingErr:         errors.New(""),
			imageTag:        "imagetag2",
			expectedBuilder: kanikoBuilder,
		},
	}

	for _, testCase := range testCases {
		controller := &controller{
			client:       fakeClient,
			dockerClient: fakeDocker,
		}
		fakeDocker.PingErr = testCase.pingErr

		builder, err := controller.createBuilder(testCase.imageConfigName, testCase.imageConf, testCase.imageTag, &testCase.options, log.Discard)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		builderAsYaml, err := yaml.Marshal(builder)
		assert.NilError(t, err, "Error marshaling builder in testCase %s", testCase.name)
		expectationAsYaml, err := yaml.Marshal(testCase.expectedBuilder)
		assert.NilError(t, err, "Error marshaling expected builder in testCase %s", testCase.name)
		assert.Equal(t, string(builderAsYaml), string(expectationAsYaml), "Unexpected cache in testCase %s", testCase.name)
		assert.Equal(t, reflect.TypeOf(builder), reflect.TypeOf(testCase.expectedBuilder), "Unexpected cache type in testCase %s", testCase.name)
	}
}
