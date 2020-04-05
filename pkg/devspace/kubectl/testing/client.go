package testing

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl"
	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/portforward"
	"github.com/devspace-cloud/devspace/pkg/util/kubeconfig"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport/spdy"
)

// Client is a fake implementation of the kubectl.Client interface
type Client struct {
	Client       kubernetes.Interface
	KubeLoader   kubeconfig.Loader
	IsKubernetes bool
	Context      string
}

// CurrentContext is a fake implementation of function
func (c *Client) CurrentContext() string {
	return c.Context
}

// KubeClient is a fake implementation of function
func (c *Client) KubeClient() kubernetes.Interface {
	return c.Client
}

// Namespace is a fake implementation of function
func (c *Client) Namespace() string {
	return "testNamespace"
}

// RestConfig is a fake implementation of function
func (c *Client) RestConfig() *rest.Config {
	return &rest.Config{
		Host: "testHost",
	}
}

// KubeConfigLoader is a fake implementation of function
func (c *Client) KubeConfigLoader() kubeconfig.Loader {
	return c.KubeLoader
}

// PrintWarning is a fake implementation of function
func (c *Client) PrintWarning(generatedConfig *generated.Config, noWarning, shouldWait bool, log log.Logger) error {
	return nil
}

// CopyFromReader is a fake implementation of function
func (c *Client) CopyFromReader(pod *k8sv1.Pod, container, containerPath string, reader io.Reader) error {
	return nil
}

// Copy is a fake implementation of function
func (c *Client) Copy(pod *k8sv1.Pod, container, containerPath, localPath string, exclude []string) error {
	return nil
}

// ExecStreamWithTransport is a fake implementation of function
func (c *Client) ExecStreamWithTransport(transport http.RoundTripper, upgrader spdy.Upgrader, pod *k8sv1.Pod, container string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer, subResource kubectl.SubResource) error {
	return nil
}

// ExecStream is a fake implementation of function
func (c *Client) ExecStream(pod *k8sv1.Pod, container string, command []string, tty bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return nil
}

// ExecBuffered is a fake implementation of function
func (c *Client) ExecBuffered(pod *k8sv1.Pod, container string, command []string, input io.Reader) ([]byte, []byte, error) {
	return []byte{}, []byte{}, nil
}

// GenericRequest is a fake implementation of function
func (c *Client) GenericRequest(options *kubectl.GenericRequestOptions) (string, error) {
	return "", nil
}

// ReadLogs is a fake implementation of function
func (c *Client) ReadLogs(namespace, podName, containerName string, lastContainerLog bool, tail *int64) (string, error) {
	return "ContainerLogs", nil
}

// LogMultipleTimeout is a fake implementation of function
func (c *Client) LogMultipleTimeout(imageSelector []string, interrupt chan error, tail *int64, writer io.Writer, timeout time.Duration, log log.Logger) error {
	_, err := writer.Write([]byte("ContainerLogs"))
	return err
}

// LogMultiple is a fake implementation of function
func (c *Client) LogMultiple(imageSelector []string, interrupt chan error, tail *int64, writer io.Writer, log log.Logger) error {
	_, err := writer.Write([]byte("ContainerLogs"))
	return err
}

// Logs is a fake implementation of function
func (c *Client) Logs(ctx context.Context, namespace, podName, containerName string, lastContainerLog bool, tail *int64, follow bool) (io.ReadCloser, error) {
	retVal := ioutil.NopCloser(strings.NewReader("ContainerLogs"))
	return retVal, nil
}

// GetUpgraderWrapper is a fake implementation of function
func (c *Client) GetUpgraderWrapper() (http.RoundTripper, kubectl.UpgraderWrapper, error) {
	return nil, nil, nil
}

// EnsureDefaultNamespace is a fake implementation of function
func (c *Client) EnsureDefaultNamespace(log log.Logger) error {
	return nil
}

// EnsureGoogleCloudClusterRoleBinding is a fake implementation of function
func (c *Client) EnsureGoogleCloudClusterRoleBinding(log log.Logger) error {
	return nil
}

// GetRunningPodsWithImage is a fake implementation of function
func (c *Client) GetRunningPodsWithImage(imageNames []string, namespace string, maxWaiting time.Duration) ([]*k8sv1.Pod, error) {
	return nil, nil
}

// GetNewestPodOnceRunning is a fake implementation of function
func (c *Client) GetNewestPodOnceRunning(labelSelector string, imageSelector []string, namespace string, maxWaiting time.Duration) (*k8sv1.Pod, error) {
	return nil, nil
}

// NewPortForwarder is a fake implementation of function
func (c *Client) NewPortForwarder(pod *k8sv1.Pod, ports []string, addresses []string, stopChan chan struct{}, readyChan chan struct{}, errorChan chan error) (*portforward.PortForwarder, error) {
	return nil, nil
}

// IsLocalKubernetes is a fake implementation of function
func (c *Client) IsLocalKubernetes() bool {
	return c.IsKubernetes
}

// FakeFakeClientset overwrites fake.Clientsets Discovery-function
type FakeFakeClientset struct {
	fake.Clientset
	RBACEnabled bool
}

// Discovery returns a fake instance of the Discovery-Interface
func (f *FakeFakeClientset) Discovery() discovery.DiscoveryInterface {
	return &FakeFakeDiscovery{
		DiscoveryInterface: f.Clientset.Discovery(),
		RBACEnabled:        f.RBACEnabled,
	}
}

// FakeFakeDiscovery overwrites FakeDiscoverys ServerResources-function
type FakeFakeDiscovery struct {
	discovery.DiscoveryInterface
	RBACEnabled bool
}

// ServerResources return one RBAC-Resource if it is enabled, else nothing
func (f *FakeFakeDiscovery) ServerResources() ([]*metav1.APIResourceList, error) {
	if f.RBACEnabled {
		return []*metav1.APIResourceList{
			&metav1.APIResourceList{
				GroupVersion: "rbac.authorization.k8s.io/v1beta1",
			},
		}, nil
	}

	return []*metav1.APIResourceList{}, nil
}
