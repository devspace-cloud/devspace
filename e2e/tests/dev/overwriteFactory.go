package dev

import (
	"github.com/loft-sh/devspace/pkg/util/log"

	"github.com/loft-sh/devspace/e2e/utils"
	"github.com/loft-sh/devspace/pkg/devspace/config/generated"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl"
	"github.com/loft-sh/devspace/pkg/devspace/services"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
)

type customFactory struct {
	*utils.BaseCustomFactory
	initialRun            bool
	imageSelectorFirstRun string
	interruptSync         chan error
	interruptPortforward  chan error
	enableSync            chan bool
}

type fakeServiceClient struct {
	services.Client
	factory *customFactory
}

// NewFakeServiceClient implements
func (c *customFactory) NewServicesClient(config *latest.Config, generated *generated.Config, kubeClient kubectl.Client, log log.Logger) services.Client {
	return &fakeServiceClient{
		Client:  services.NewClient(config, generated, kubeClient, log),
		factory: c,
	}
}

func (s *fakeServiceClient) StartPortForwarding(interrupt chan error) error {
	return s.Client.StartPortForwarding(s.factory.interruptPortforward)
}

func (s *fakeServiceClient) StartSync(interrupt chan error, printSync, verboseSync bool) error {
	// wait for it
	_, notClosed := <-s.factory.enableSync
	if notClosed {
		close(s.factory.enableSync)
	}

	return s.Client.StartSync(s.factory.interruptSync, printSync, verboseSync)
}

func (s *fakeServiceClient) StartTerminal(options targetselector.Options, args []string, workDir string, interrupt chan error, wait bool) (int, error) {
	return s.Client.StartTerminal(options, args, workDir, interrupt, wait)
}
