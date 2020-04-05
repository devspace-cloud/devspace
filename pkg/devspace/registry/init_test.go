package registry

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	fakedocker "github.com/devspace-cloud/devspace/pkg/devspace/docker/testing"
	kubectl "github.com/devspace-cloud/devspace/pkg/devspace/kubectl/testing"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/ptr"

	dockertypes "github.com/docker/docker/api/types"
	"gotest.tools/assert"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type createPullSecretTestCase struct {
	name string

	namespace       string
	serviceAccounts map[string][]string
	secrets         []k8sv1.Secret
	imagesInConfig  map[string]*latest.ImageConfig

	expectedErr                          string
	expectedPullSecretsInServiceAccounts map[string][]string
	expectedSecrets                      map[string]k8sv1.Secret
}

func TestCreatePullSecrets(t *testing.T) {
	testCases := []createPullSecretTestCase{
		createPullSecretTestCase{
			name:            "One simple creation without default service account",
			namespace:       "testNamespace",
			secrets:         []k8sv1.Secret{{ObjectMeta: metav1.ObjectMeta{Name: "devspace-auth-docker"}}},
			serviceAccounts: map[string][]string{"default": {"secretDefault", "devspace-auth-docker"}},
			imagesInConfig: map[string]*latest.ImageConfig{
				"testimage": &latest.ImageConfig{
					CreatePullSecret: ptr.Bool(true),
					Image:            "testimage",
				},
				"testimage2": &latest.ImageConfig{
					CreatePullSecret: ptr.Bool(true),
					Image:            "hub.docker.com/user/myimage",
				},
			},
			expectedPullSecretsInServiceAccounts: map[string][]string{
				"default": {"secretDefault", "devspace-auth-docker", "devspace-auth-hub-docker-com"},
			},
			expectedSecrets: map[string]k8sv1.Secret{
				"devspace-auth-docker": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "devspace-auth-docker",
						Namespace: "testNamespace",
					},
					Data: map[string][]byte{
						k8sv1.DockerConfigJsonKey: []byte(`{
			"auths": {
				"https://index.docker.io/v1/": {
					"auth": "` + base64.StdEncoding.EncodeToString([]byte("user:pass")) + `",
					"email": "noreply@devspace.cloud"
				}
			}
		}`),
					},
					Type: k8sv1.SecretTypeDockerConfigJson,
				},
				"devspace-auth-hub-docker-com": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "devspace-auth-hub-docker-com",
						Namespace: "testNamespace",
					},
					Data: map[string][]byte{
						k8sv1.DockerConfigJsonKey: []byte(`{
			"auths": {
				"https://index.docker.io/v1/": {
					"auth": "` + base64.StdEncoding.EncodeToString([]byte("user:pass")) + `",
					"email": "noreply@devspace.cloud"
				}
			}
		}`),
					},
					Type: k8sv1.SecretTypeDockerConfigJson,
				},
			},
		},
	}

	for _, testCase := range testCases {
		//Setting up kubeClient
		kubeClient := &kubectl.Client{
			Client: fake.NewSimpleClientset(),
		}
		_, err := kubeClient.Client.CoreV1().Namespaces().Create(&k8sv1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testCase.namespace,
			},
		})
		assert.NilError(t, err, "Error creating namespace in testCase %s", testCase.name)

		for name, secrets := range testCase.serviceAccounts {
			imagePullSecrets := []k8sv1.LocalObjectReference{}
			for _, secret := range secrets {
				imagePullSecrets = append(imagePullSecrets, k8sv1.LocalObjectReference{Name: secret})
			}
			_, err = kubeClient.Client.CoreV1().ServiceAccounts(testCase.namespace).Create(&k8sv1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: testCase.namespace,
				},
				ImagePullSecrets: imagePullSecrets,
			})
		}

		for _, obj := range testCase.secrets {
			_, err = kubeClient.Client.CoreV1().Secrets(testCase.namespace).Create(&obj)
		}

		// Create fake devspace config
		testConfig := &latest.Config{
			Images:      testCase.imagesInConfig,
			Deployments: []*latest.DeploymentConfig{{}},
		}

		client := &client{
			config:     testConfig,
			kubeClient: kubeClient,
			dockerClient: &fakedocker.FakeClient{
				AuthConfig: &dockertypes.AuthConfig{
					Username: "user",
					Password: "pass",
				},
			},
			log: log.Discard,
		}

		err = client.CreatePullSecrets()

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error creating pull secrets in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error creating pull secrets in testCase %s", testCase.name)
		}

		for saName, expectedSecrets := range testCase.expectedPullSecretsInServiceAccounts {
			sa, err := kubeClient.Client.CoreV1().ServiceAccounts(testCase.namespace).Get(saName, metav1.GetOptions{})
			assert.NilError(t, err, "Unexpected error getting serviceaccount %s in testCase %s", saName, testCase.name)
			expectedImagePullSecrets := []k8sv1.LocalObjectReference{}
			for _, secret := range expectedSecrets {
				expectedImagePullSecrets = append(expectedImagePullSecrets, k8sv1.LocalObjectReference{Name: secret})
			}
			assert.Assert(t, reflect.DeepEqual(sa.ImagePullSecrets, expectedImagePullSecrets), "Unexpected secrets in sericeAccount %s in testCase %s", saName, testCase.name)
		}

		for expectedSecretName, expectedSecretObj := range testCase.expectedSecrets {
			secret, err := kubeClient.Client.CoreV1().Secrets(testCase.namespace).Get(expectedSecretName, metav1.GetOptions{})
			assert.NilError(t, err, "Unexpected error getting secret %s in testCase %s", expectedSecretName, testCase.name)
			t.Log(string(secret.Data[".dockerconfigjson"]))
			t.Log(string(expectedSecretObj.Data[".dockerconfigjson"]))
			assert.Assert(t, reflect.DeepEqual(*secret, expectedSecretObj), "Unexpected secret %s in testCase %s", expectedSecretName, testCase.name)
		}
	}

}
