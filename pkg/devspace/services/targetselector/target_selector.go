package targetselector

import (
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultPodQuestion defines the default question for selecting a pod
const DefaultPodQuestion = "Select a pod"

// DefaultContainerQuestion defines the default question for selecting a container
const DefaultContainerQuestion = "Select a container"

// TargetSelector is the struct that will select a target
type TargetSelector struct {
	PodQuestion       *string
	ContainerQuestion *string

	AllowNonRunning bool
	SkipWait        bool

	namespace string
	pick      bool

	labelSelector string
	imageSelector []string
	podName       string
	containerName string

	allowPick bool

	kubeClient kubectl.Client
	config     *latest.Config
}

// NewTargetSelector creates a new target selector for selecting a target pod or container
func NewTargetSelector(config *latest.Config, kubeClient kubectl.Client, sp *SelectorParameter, allowPick bool, imageSelector []string) (*TargetSelector, error) {
	// Get namespace
	namespace, err := sp.GetNamespace(config, kubeClient)
	if err != nil {
		return nil, err
	}

	// Get label selector
	labelSelector, err := sp.GetLabelSelector(config)
	if err != nil {
		return nil, err
	}

	return &TargetSelector{
		namespace:     namespace,
		labelSelector: labelSelector,
		imageSelector: imageSelector,
		podName:       sp.GetPodName(),
		containerName: sp.GetContainerName(),
		pick:          allowPick && sp.CmdParameter.Pick != nil && *sp.CmdParameter.Pick == true,

		kubeClient: kubeClient,
		allowPick:  allowPick,
		config:     config,
	}, nil
}

// GetPod retrieves a pod
func (t *TargetSelector) GetPod(log log.Logger) (*v1.Pod, error) {
	if t.pick == false {
		timeout := time.Minute * 10
		if t.SkipWait == true {
			timeout = 0
		}

		if t.podName != "" {
			pod, err := t.kubeClient.KubeClient().CoreV1().Pods(t.namespace).Get(t.podName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}

			podStatus := kubectl.GetPodStatus(pod)
			if podStatus != "Running" && strings.HasPrefix(podStatus, "Init") == false {
				return nil, errors.Errorf(message.PodStatusCritical, pod.Name, podStatus, pod.Name)
			}

			return pod, nil
		} else if t.labelSelector != "" {
			pod, err := t.kubeClient.GetNewestPodOnceRunning(t.labelSelector, t.imageSelector, t.namespace, timeout)
			if err != nil {
				return nil, err
			}

			return pod, nil
		} else if len(t.imageSelector) > 0 {
			// Retrieve pods running with that image
			pods, err := t.kubeClient.GetRunningPodsWithImage(t.imageSelector, t.namespace, timeout)
			if err != nil {
				return nil, err
			}

			// Take first pod if only one is found
			if len(pods) == 1 {
				return pods[0], nil
			} else if len(pods) == 0 {
				return nil, errors.Errorf("Couldn't find a running pod with image selector '%s'", strings.Join(t.imageSelector, ", "))
			}

			if len(pods) == 0 {
				return nil, errors.Errorf("Couldn't find a running pod with image selector '%s'", strings.Join(t.imageSelector, ", "))
			}

			// Show picker if allowed
			if t.allowPick {
				podNames := []string{}
				podMap := map[string]*v1.Pod{}
				for _, pod := range pods {
					podNames = append(podNames, pod.Name)
					podMap[pod.Name] = pod
				}

				podName, err := log.Question(&survey.QuestionOptions{
					Question: *t.PodQuestion,
					Options:  podNames,
				})
				if err != nil {
					return nil, err
				}

				return podMap[podName], nil
			}

			log.Warnf("Multiple pods with image selector '%s' found. Using first pod found", strings.Join(t.imageSelector, ", "))
			return pods[0], nil
		}
	}

	// Don't allow pick
	if t.allowPick == false {
		return nil, errors.New("Couldn't find a running pod, because no labelselector or pod name was specified")
	}

	// Ask for pod
	pod, err := SelectPod(t.kubeClient, t.namespace, nil, t.PodQuestion, !t.AllowNonRunning, log)
	if err != nil {
		return nil, err
	}

	return pod, nil
}

const initContainerOptionPrefix = "Init: "

// GetContainer retrieves a container and pod
func (t *TargetSelector) GetContainer(allowInitContainer bool, log log.Logger) (*v1.Pod, *v1.Container, error) {
	pod, err := t.GetPod(log)
	if err != nil {
		return nil, nil, err
	} else if pod == nil {
		return nil, nil, errors.Errorf("Couldn't find a running pod in namespace %s", t.namespace)
	}

	// Check if we allow selecting also init containers
	if allowInitContainer && len(pod.Spec.InitContainers) > 0 {
		if len(pod.Spec.Containers) == 0 && len(pod.Spec.InitContainers) == 1 {
			return pod, &pod.Spec.InitContainers[0], nil
		}

		if t.pick == false && t.containerName != "" {
			// Find init container
			for _, container := range pod.Spec.InitContainers {
				if container.Name == t.containerName {
					return pod, &container, nil
				}
			}
			for _, container := range pod.Spec.Containers {
				if container.Name == t.containerName {
					return pod, &container, nil
				}
			}

			return nil, nil, errors.Errorf("Couldn't find container %s in pod %s", t.containerName, pod.Name)
		} else if len(t.imageSelector) > 0 {
			// TODO: What happens if there are 2 containers with the same image?
			for _, container := range pod.Spec.InitContainers {
				for _, imageName := range t.imageSelector {
					if imageName == container.Image {
						return pod, &container, nil
					}
				}
			}
			for _, container := range pod.Spec.Containers {
				for _, imageName := range t.imageSelector {
					if imageName == container.Image {
						return pod, &container, nil
					}
				}
			}
		}

		// Don't allow pick
		if t.allowPick == false {
			return nil, nil, errors.Errorf("Couldn't select a container in pod %s, because no container name was specified", pod.Name)
		}

		options := []string{}
		for _, container := range pod.Spec.InitContainers {
			options = append(options, initContainerOptionPrefix+container.Name)
		}
		for _, container := range pod.Spec.Containers {
			options = append(options, container.Name)
		}

		if t.ContainerQuestion == nil {
			t.ContainerQuestion = ptr.String(DefaultContainerQuestion)
		}

		containerName, err := log.Question(&survey.QuestionOptions{
			Question: *t.ContainerQuestion,
			Options:  options,
		})
		if err != nil {
			return nil, nil, err
		} else if strings.HasPrefix(containerName, initContainerOptionPrefix) {
			containerName = containerName[len(initContainerOptionPrefix):]
		}

		for _, container := range pod.Spec.InitContainers {
			if container.Name == containerName {
				return pod, &container, nil
			}
		}
		for _, container := range pod.Spec.Containers {
			if container.Name == containerName {
				return pod, &container, nil
			}
		}
	}

	// Select container if necessary
	if pod.Spec.Containers != nil && len(pod.Spec.Containers) == 1 {
		return pod, &pod.Spec.Containers[0], nil
	} else if pod.Spec.Containers != nil && len(pod.Spec.Containers) > 1 {
		if t.pick == false && t.containerName != "" {
			// Find container
			for _, container := range pod.Spec.Containers {
				if container.Name == t.containerName {
					return pod, &container, nil
				}
			}

			return nil, nil, errors.Errorf("Couldn't find container %s in pod %s", t.containerName, pod.Name)
		} else if len(t.imageSelector) > 0 {
			// TODO: What happens if there are 2 containers with the same image?
			for _, container := range pod.Spec.Containers {
				for _, imageName := range t.imageSelector {
					if imageName == container.Image {
						return pod, &container, nil
					}
				}
			}
		}

		// Don't allow pick
		if t.allowPick == false {
			return nil, nil, errors.Errorf("Couldn't select a container in pod %s, because no container name was specified", pod.Name)
		}

		options := []string{}
		for _, container := range pod.Spec.Containers {
			options = append(options, container.Name)
		}

		if t.ContainerQuestion == nil {
			t.ContainerQuestion = ptr.String(DefaultContainerQuestion)
		}

		containerName, err := log.Question(&survey.QuestionOptions{
			Question: *t.ContainerQuestion,
			Options:  options,
		})
		if err != nil {
			return nil, nil, err
		}

		for _, container := range pod.Spec.Containers {
			if container.Name == containerName {
				return pod, &container, nil
			}
		}
	}

	return pod, nil, nil
}

func ImageSelectorFromConfig(configImageName string, config *latest.Config, generated *generated.Config) []string {
	var imageSelector []string
	if configImageName != "" && generated != nil && config != nil {
		imageConfigCache := generated.GetActive().GetImageCache(configImageName)
		if imageConfigCache.ImageName != "" {
			imageSelector = []string{imageConfigCache.ImageName + ":" + imageConfigCache.Tag}
		} else if config.Images[configImageName] != nil {
			if len(config.Images[configImageName].Tags) > 0 {
				imageSelector = []string{config.Images[configImageName].Image + ":" + config.Images[configImageName].Tags[0]}
			} else {
				imageSelector = []string{config.Images[configImageName].Image}
			}
		}
	}

	return imageSelector
}
