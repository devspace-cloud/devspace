package kubectl

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/util"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/portforward"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/survey"

	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/transport/spdy"
)

// Client holds all kubect functions
type Client interface {
	CurrentContext() string
	KubeClient() kubernetes.Interface
	Namespace() string
	RestConfig() *rest.Config
	KubeConfigLoader() kubeconfig.Loader

	PrintWarning(generatedConfig *generated.Config, noWarning, shouldWait bool, log log.Logger) error
	CopyFromReader(pod *k8sv1.Pod, container, containerPath string, reader io.Reader) error
	Copy(pod *k8sv1.Pod, container, containerPath, localPath string, exclude []string) error

	ExecStreamWithTransport(transport http.RoundTripper, upgrader spdy.Upgrader, pod *k8sv1.Pod, container string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer, subResource SubResource) error
	ExecStream(pod *k8sv1.Pod, container string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) error
	ExecBuffered(pod *k8sv1.Pod, container string, command []string, input io.Reader) ([]byte, []byte, error)

	GenericRequest(options *GenericRequestOptions) (string, error)

	ReadLogs(namespace, podName, containerName string, lastContainerLog bool, tail *int64) (string, error)
	LogMultipleTimeout(imageSelector []string, interrupt chan error, tail *int64, writer io.Writer, timeout time.Duration, log log.Logger) error
	LogMultiple(imageSelector []string, interrupt chan error, tail *int64, writer io.Writer, log log.Logger) error
	Logs(ctx context.Context, namespace, podName, containerName string, lastContainerLog bool, tail *int64, follow bool) (io.ReadCloser, error)

	GetUpgraderWrapper() (http.RoundTripper, UpgraderWrapper, error)

	EnsureDefaultNamespace(log log.Logger) error
	EnsureGoogleCloudClusterRoleBinding(log log.Logger) error
	GetRunningPodsWithImage(imageNames []string, namespace string, maxWaiting time.Duration) ([]*k8sv1.Pod, error)
	GetNewestPodOnceRunning(labelSelector string, imageSelector []string, namespace string, maxWaiting time.Duration) (*k8sv1.Pod, error)
	NewPortForwarder(pod *k8sv1.Pod, ports []string, addresses []string, stopChan chan struct{}, readyChan chan struct{}, errorChan chan error) (*portforward.PortForwarder, error)
	IsLocalKubernetes() bool
}

type client struct {
	Client       kubernetes.Interface
	ClientConfig clientcmd.ClientConfig
	restConfig   *rest.Config
	kubeLoader   kubeconfig.Loader

	currentContext string
	namespace      string
}

// NewDefaultClient creates the new default kube client from the active context @Factory
func NewDefaultClient() (Client, error) {
	return NewClientFromContext("", "", false, kubeconfig.NewLoader())
}

// NewClientFromContext creates a new kubernetes client from given context @Factory
func NewClientFromContext(context, namespace string, switchContext bool, kubeLoader kubeconfig.Loader) (Client, error) {
	// Load new raw config
	kubeConfigOriginal, err := kubeLoader.LoadRawConfig()
	if err != nil {
		return nil, err
	}

	// We clone the config here to avoid changing the single loaded config
	kubeConfig := clientcmdapi.Config{}
	err = util.Convert(&kubeConfigOriginal, &kubeConfig)
	if err != nil {
		return nil, err
	}

	if len(kubeConfig.Clusters) == 0 {
		return nil, errors.Errorf("kube config is invalid: please make sure you have an existing valid kube config")
	}

	// If we should use a certain kube context use that
	var (
		activeContext   = kubeConfig.CurrentContext
		activeNamespace = metav1.NamespaceDefault
		saveConfig      = false
	)

	// Set active context
	if context != "" && activeContext != context {
		activeContext = context
		if switchContext {
			kubeConfig.CurrentContext = activeContext
			saveConfig = true
		}
	}

	// Set active namespace
	if kubeConfig.Contexts[activeContext] != nil {
		if kubeConfig.Contexts[activeContext].Namespace != "" {
			activeNamespace = kubeConfig.Contexts[activeContext].Namespace
		}

		if namespace != "" && activeNamespace != namespace {
			activeNamespace = namespace
			kubeConfig.Contexts[activeContext].Namespace = activeNamespace
			if switchContext {
				saveConfig = true
			}
		}
	}

	// Should we save the kube config?
	if saveConfig {
		err = kubeLoader.SaveConfig(&kubeConfig)
		if err != nil {
			return nil, errors.Errorf("Error saving kube config: %v", err)
		}
	}

	clientConfig := clientcmd.NewNonInteractiveClientConfig(kubeConfig, activeContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules())
	if kubeConfig.Contexts[activeContext] == nil {
		return nil, errors.Errorf("Error loading kube config, context '%s' doesn't exist", activeContext)
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "new client")
	}

	return &client{
		Client:       kubeClient,
		ClientConfig: clientConfig,
		restConfig:   restConfig,
		kubeLoader:   kubeLoader,

		namespace:      activeNamespace,
		currentContext: activeContext,
	}, nil
}

// NewClientBySelect creates a new kubernetes client by user select @Factory
func NewClientBySelect(allowPrivate bool, switchContext bool, kubeLoader kubeconfig.Loader, log log.Logger) (Client, error) {
	kubeConfig, err := kubeLoader.LoadRawConfig()
	if err != nil {
		return nil, err
	}

	// Get all kube contexts
	options := make([]string, 0, len(kubeConfig.Contexts))
	for context := range kubeConfig.Contexts {
		options = append(options, context)
	}
	if len(options) == 0 {
		return nil, errors.New("No kubectl context found. Make sure kubectl is installed and you have a working kubernetes context configured")
	}

	sort.Strings(options)
	for true {
		kubeContext, err := log.Question(&survey.QuestionOptions{
			Question:     "Which kube context do you want to use",
			DefaultValue: kubeConfig.CurrentContext,
			Options:      options,
		})
		if err != nil {
			return nil, err
		}

		// Check if cluster is in private network
		if allowPrivate == false {
			context := kubeConfig.Contexts[kubeContext]
			cluster := kubeConfig.Clusters[context.Cluster]

			url, err := url.Parse(cluster.Server)
			if err != nil {
				return nil, errors.Wrap(err, "url parse")
			}

			ip := net.ParseIP(url.Hostname())
			if ip != nil {
				if IsPrivateIP(ip) {
					log.Infof("Clusters with private ips (%s) cannot be used", url.Hostname())
					continue
				}
			}
		}

		return NewClientFromContext(kubeContext, "", switchContext, kubeLoader)
	}

	return nil, errors.New("We should not reach this point")
}

// This constant is used as a variable so it can be overwritten in a test
var second = time.Second

// PrintWarning prints a warning if the last kube context is different than this one
func (client *client) PrintWarning(generatedConfig *generated.Config, noWarning, shouldWait bool, log log.Logger) error {
	if generatedConfig != nil && log.GetLevel() >= logrus.InfoLevel && noWarning == false {
		// print warning if context or namespace has changed since last deployment process (expect if explicitly provided as flags)
		if generatedConfig.GetActive().LastContext != nil {
			wait := false

			if generatedConfig.GetActive().LastContext.Context != "" && generatedConfig.GetActive().LastContext.Context != client.currentContext {
				log.WriteString("\n")
				log.Warnf(ansi.Color("Are you using the correct kube context?", "white+b"))
				log.Warnf("Current kube context: '%s'", ansi.Color(client.currentContext, "white+b"))
				log.Warnf("Last    kube context: '%s'", ansi.Color(generatedConfig.GetActive().LastContext.Context, "white+b"))
				log.WriteString("\n")

				log.Infof("Use the '%s' flag to switch to the context and namespace previously used to deploy this project", ansi.Color("-s / --switch-context", "white+b"))
				log.Infof("Or use the '%s' flag to ignore this warning", ansi.Color("--no-warn", "white+b"))
				wait = true
			} else if generatedConfig.GetActive().LastContext.Namespace != "" && generatedConfig.GetActive().LastContext.Namespace != client.namespace {
				log.WriteString("\n")
				log.Warnf(ansi.Color("Are you using the correct namespace?", "white+b"))
				log.Warnf("Current namespace: '%s'", ansi.Color(client.namespace, "white+b"))
				log.Warnf("Last    namespace: '%s'", ansi.Color(generatedConfig.GetActive().LastContext.Namespace, "white+b"))
				log.WriteString("\n")

				log.Infof("Use the '%s' flag to switch to the context and namespace previously used to deploy this project", ansi.Color("-s / --switch-context", "white+b"))
				log.Infof("Or use the '%s' flag to ignore this warning", ansi.Color("--no-warn", "white+b"))
				wait = true
			}

			if wait && shouldWait {
				log.StartWait("Will continue in 10 seconds...")
				time.Sleep(10 * second)
				log.StopWait()
				log.WriteString("\n")
			}
		}

		// Warn if using default namespace unless previous deployment was also to default namespace
		if shouldWait && client.namespace == metav1.NamespaceDefault && (generatedConfig.GetActive().LastContext == nil || generatedConfig.GetActive().LastContext.Namespace != metav1.NamespaceDefault) {
			log.Warn("Deploying into the 'default' namespace is usually not a good idea as this namespace cannot be deleted\n")
			log.StartWait("Will continue in 5 seconds...")
			time.Sleep(5 * second)
			log.StopWait()
		}
	}

	// Info messages
	log.Infof("Using kube context '%s'", ansi.Color(client.currentContext, "white+b"))
	log.Infof("Using namespace '%s'", ansi.Color(client.namespace, "white+b"))

	return nil
}

func (client *client) CurrentContext() string {
	return client.currentContext
}

func (client *client) KubeClient() kubernetes.Interface {
	return client.Client
}

func (client *client) Namespace() string {
	return client.namespace
}

func (client *client) RestConfig() *rest.Config {
	return client.restConfig
}

func (client *client) KubeConfigLoader() kubeconfig.Loader {
	return client.kubeLoader
}
