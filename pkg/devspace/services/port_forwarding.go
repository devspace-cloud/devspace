package services

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/devspace/pkg/devspace/config/generated"

	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/services/targetselector"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/loft-sh/devspace/pkg/util/port"
	"github.com/pkg/errors"
)

// StartPortForwarding starts the port forwarding functionality
func (serviceClient *client) StartPortForwarding(interrupt chan error) error {
	if serviceClient.config.Dev == nil {
		return nil
	}

	var cache *generated.CacheConfig
	if serviceClient.generated != nil {
		cache = serviceClient.generated.GetActive()
	}

	for _, portForwarding := range serviceClient.config.Dev.Ports {
		if len(portForwarding.PortMappings) == 0 {
			continue
		}

		// start port forwarding
		err := serviceClient.startForwarding(cache, portForwarding, interrupt, serviceClient.log)
		if err != nil {
			return err
		}
	}

	return nil
}

func (serviceClient *client) startForwarding(cache *generated.CacheConfig, portForwarding *latest.PortForwardingConfig, interrupt chan error, log logpkg.Logger) error {
	// apply config & set image selector
	options := targetselector.NewEmptyOptions().ApplyConfigParameter(portForwarding.LabelSelector, portForwarding.Namespace, "", "")
	options.AllowPick = false
	options.ImageSelector = targetselector.ImageSelectorFromConfig(portForwarding.ImageName, serviceClient.config, cache)
	options.WaitingStrategy = targetselector.NewUntilNewestRunningWaitingStrategy(time.Second * 2)
	options.SkipInitContainers = true

	// start port forwarding
	log.StartWait("Port-Forwarding: Waiting for containers to start...")
	pod, err := targetselector.NewTargetSelector(serviceClient.client).SelectSinglePod(context.TODO(), options, log)
	log.StopWait()
	if err != nil {
		return errors.Errorf("%s: %s", message.SelectorErrorPod, err.Error())
	} else if pod == nil {
		return nil
	}

	ports := make([]string, len(portForwarding.PortMappings))
	addresses := make([]string, len(portForwarding.PortMappings))
	for index, value := range portForwarding.PortMappings {
		if value.LocalPort == nil {
			return errors.Errorf("port is not defined in portmapping %d", index)
		}

		localPort := strconv.Itoa(*value.LocalPort)
		remotePort := localPort
		if value.RemotePort != nil {
			remotePort = strconv.Itoa(*value.RemotePort)
		}

		open, _ := port.Check(*value.LocalPort)
		if open == false {
			serviceClient.log.Warnf("Seems like port %d is already in use. Is another application using that port?", *value.LocalPort)
		}

		ports[index] = localPort + ":" + remotePort
		if value.BindAddress == "" {
			addresses[index] = "localhost"
		} else {
			addresses[index] = value.BindAddress
		}
	}

	readyChan := make(chan struct{})
	errorChan := make(chan error)

	pf, err := serviceClient.client.NewPortForwarder(pod, ports, addresses, make(chan struct{}), readyChan, errorChan)
	if err != nil {
		return errors.Errorf("Error starting port forwarding: %v", err)
	}

	go func() {
		err := pf.ForwardPorts()
		if err != nil {
			serviceClient.log.Fatalf("Error forwarding ports: %v", err)
		}
	}()

	// Wait till forwarding is ready
	select {
	case <-readyChan:
		log.Donef("Port forwarding started on %s (%s/%s)", strings.Join(ports, ", "), pod.Namespace, pod.Name)
	case <-time.After(20 * time.Second):
		return errors.Errorf("Timeout waiting for port forwarding to start")
	}

	go func(portForwarding *latest.PortForwardingConfig, interrupt chan error) {
		select {
		case err := <-errorChan:
			if err != nil {
				pf.Close()
				for {
					err = serviceClient.startForwarding(cache, portForwarding, interrupt, logpkg.Discard)
					if err != nil {
						serviceClient.log.Errorf("Error restarting port-forwarding: %v", err)
						serviceClient.log.Errorf("Will try again in 3 seconds")
						continue
					}

					time.Sleep(time.Second * 3)
					break
				}
			}
		case <-interrupt:
			pf.Close()
		}
	}(portForwarding, interrupt)

	return nil
}
