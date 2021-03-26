package initcmd

import (
	dockertypes "github.com/docker/docker/api/types"
	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/configure"
	"github.com/loft-sh/devspace/pkg/devspace/docker"
	fakedocker "github.com/loft-sh/devspace/pkg/devspace/docker/testing"
	"github.com/loft-sh/devspace/pkg/util/log"
)

type customFactory struct {
	*utils.BaseCustomFactory
}

func (c *customFactory) NewDockerClientWithMinikube(currentKubeContext string, preferMinikube bool, logger log.Logger) (docker.Client, error) {
	fakeDockerClient := &fakedocker.FakeClient{
		AuthConfig: &dockertypes.AuthConfig{
			Username: "user",
			Password: "pass",
		},
	}
	return fakeDockerClient, nil
}

// NewConfigureManager implements interface
func (f *customFactory) NewConfigureManager(config *latest.Config, log log.Logger) configure.Manager {
	return configure.NewManager(f, config, log)
}
