package kubectl

import (
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	fakekubeloader "github.com/devspace-cloud/devspace/pkg/util/kubeconfig/testing"
	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	fakesurvey "github.com/devspace-cloud/devspace/pkg/util/survey/testing"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type newClientFromContextTestCase struct {
	name string

	clusters       map[string]*clientcmdapi.Cluster
	context        string
	namespace      string
	switchContext  bool
	contexts       map[string]*clientcmdapi.Context
	currentContext string

	expectedErr    bool
	expectedClient *client
}

func TestNewClientFromContext(t *testing.T) {
	testCases := []newClientFromContextTestCase{
		{
			name:        "no clusters",
			expectedErr: true,
		},
		{
			name:    "context not there",
			context: "notThere",
			clusters: map[string]*clientcmdapi.Cluster{
				"": {},
			},
			currentContext: "current",
			expectedErr:    true,
		},
		{
			name:    "Create successfully",
			context: "context1",
			clusters: map[string]*clientcmdapi.Cluster{
				"": {
					Server: "someServer",
				},
			},
			contexts: map[string]*clientcmdapi.Context{
				"context1": {
					Namespace: "contextNamespace",
				},
			},
			switchContext:  true,
			currentContext: "current",
			namespace:      "paramNS",
			expectedClient: &client{
				Client: &kubernetes.Clientset{
					DiscoveryClient: &discovery.DiscoveryClient{
						LegacyPrefix: "/api",
					},
				},
				ClientConfig: &clientcmd.DirectClientConfig{},
				restConfig: &rest.Config{
					Host: "someServer",
				},
				currentContext: "context1",
				namespace:      "paramNS",
			},
		},
	}

	for _, testCase := range testCases {
		kubeLoader := &fakekubeloader.Loader{
			RawConfig: &clientcmdapi.Config{
				Clusters:       testCase.clusters,
				Contexts:       testCase.contexts,
				CurrentContext: testCase.currentContext,
			},
		}
		result, err := NewClientFromContext(testCase.context, testCase.namespace, testCase.switchContext, kubeLoader)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		clientAsYaml, err := yaml.Marshal(result)
		assert.NilError(t, err, "Error parsing client to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedClient)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(clientAsYaml), string(expectedAsYaml), "Unexpected client in testCase %s", testCase.name)
		if testCase.expectedClient != nil {
			cl := result.(*client)
			assert.Equal(t, cl.restConfig.ServerName, testCase.expectedClient.restConfig.ServerName, "Unexpected client server in testCase %s", testCase.name)
			assert.Equal(t, cl.currentContext, testCase.expectedClient.currentContext, "Unexpected client in testCase %s", testCase.name)
			assert.Equal(t, cl.namespace, testCase.expectedClient.namespace, "Unexpected client in testCase %s", testCase.name)
		}
	}
}

type newClientBySelectTestCase struct {
	name string

	allowPrivate   bool
	switchContext  bool
	answers        []string
	clusters       map[string]*clientcmdapi.Cluster
	context        string
	namespace      string
	contexts       map[string]*clientcmdapi.Context
	currentContext string

	expectedErr    bool
	expectedClient *client
}

func TestNewClientBySelect(t *testing.T) {
	testCases := []newClientBySelectTestCase{
		{
			name:        "no contexts",
			expectedErr: true,
		},
		{
			name:    "Create successfully",
			context: "context1",
			clusters: map[string]*clientcmdapi.Cluster{
				"cluster1": {
					Server: "someServer",
				},
				"private": {
					Server: "http://127.0.0.1:80/path",
				},
			},
			contexts: map[string]*clientcmdapi.Context{
				"context1": {
					Cluster:   "cluster1",
					Namespace: "contextNamespace",
				},
				"private": {
					Cluster: "private",
				},
			},
			currentContext: "current",
			switchContext:  true,
			expectedClient: &client{
				Client: &kubernetes.Clientset{
					DiscoveryClient: &discovery.DiscoveryClient{
						LegacyPrefix: "/api",
					},
				},
				ClientConfig: &clientcmd.DirectClientConfig{},
				restConfig: &rest.Config{
					Host: "someServer",
				},
				currentContext: "context1",
				namespace:      "contextNamespace",
			},
			answers: []string{"private", "context1"},
		},
	}

	for _, testCase := range testCases {
		kubeLoader := &fakekubeloader.Loader{
			RawConfig: &clientcmdapi.Config{
				Clusters:       testCase.clusters,
				Contexts:       testCase.contexts,
				CurrentContext: testCase.currentContext,
			},
		}
		logger := &log.FakeLogger{
			Survey: fakesurvey.NewFakeSurvey(),
		}
		for _, answer := range testCase.answers {
			logger.Survey.SetNextAnswer(answer)
		}
		result, err := NewClientBySelect(testCase.allowPrivate, testCase.switchContext, kubeLoader, logger)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		clientAsYaml, err := yaml.Marshal(result)
		assert.NilError(t, err, "Error parsing client to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedClient)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(clientAsYaml), string(expectedAsYaml), "Unexpected client in testCase %s", testCase.name)
		if testCase.expectedClient != nil {
			cl := result.(*client)
			assert.Equal(t, cl.restConfig.ServerName, testCase.expectedClient.restConfig.ServerName, "Unexpected client server in testCase %s", testCase.name)
			assert.Equal(t, cl.currentContext, testCase.expectedClient.currentContext, "Unexpected client in testCase %s", testCase.name)
			assert.Equal(t, cl.namespace, testCase.expectedClient.namespace, "Unexpected client in testCase %s", testCase.name)
		}
	}
}

type printWarningTestCase struct {
	name string

	generatedConfig *generated.Config
	noWarning       bool
	shouldWait      bool
	clientNamespace string
	clientContext   string

	expectedErr bool
}

func TestPrintWarning(t *testing.T) {
	testCases := []printWarningTestCase{
		{
			name: "Last context is different than current",
			generatedConfig: &generated.Config{
				ActiveProfile: "active",
				Profiles: map[string]*generated.CacheConfig{
					"active": &generated.CacheConfig{
						LastContext: &generated.LastContextConfig{
							Context: "someContext",
						},
					},
				},
			},
			shouldWait:      true,
			clientNamespace: metav1.NamespaceDefault,
		},
		{
			name: "Last namespace is different than current",
			generatedConfig: &generated.Config{
				ActiveProfile: "active",
				Profiles: map[string]*generated.CacheConfig{
					"active": &generated.CacheConfig{
						LastContext: &generated.LastContextConfig{
							Namespace: "someNs",
						},
					},
				},
			},
		},
	}

	second = 0
	defer func() { second = time.Second }()

	for _, testCase := range testCases {
		client := &client{
			namespace:      testCase.clientNamespace,
			currentContext: testCase.clientContext,
		}
		err := client.PrintWarning(testCase.generatedConfig, testCase.noWarning, testCase.shouldWait, &log.FakeLogger{Level: logrus.InfoLevel})

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}
	}
}
