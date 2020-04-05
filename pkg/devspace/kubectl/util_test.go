package kubectl

import (
	"net"
	"testing"
	"time"

	log "github.com/devspace-cloud/devspace/pkg/util/log/testing"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type isPrivateIPTestCase struct {
	name string

	ip string

	expectedIsPrivate bool
}

func TestIsPrivateIP(t *testing.T) {
	testCases := []isPrivateIPTestCase{
		{
			name:              "Private IP",
			ip:                "127.0.0.0/8",
			expectedIsPrivate: true,
		},
		{
			name:              "Google IP",
			ip:                "64.233.160.0",
			expectedIsPrivate: false,
		},
	}

	for _, testCase := range testCases {
		ip, _, _ := net.ParseCIDR(testCase.ip)
		assert.Equal(t, IsPrivateIP(ip), testCase.expectedIsPrivate, "Unexpected resut in testCase %s", testCase.name)
	}
}

type ensureDefaultNamespaceTestCase struct {
	name string

	namespace  string
	namespaces []string

	expectedErr        bool
	expectedNamespaces []string
}

func TestEnsureDefaultNamespace(t *testing.T) {
	testCases := []ensureDefaultNamespaceTestCase{
		{
			name:               "Namespace is already there",
			namespace:          "exists",
			namespaces:         []string{"exists"},
			expectedNamespaces: []string{"exists"},
		},
		{
			name:               "Namespace is new",
			namespace:          "new",
			namespaces:         []string{"exists"},
			expectedNamespaces: []string{"exists", "new"},
		},
	}

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		for _, namespace := range testCase.namespaces {
			kubeClient.CoreV1().Namespaces().Create(&k8sv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			})
		}
		client := &client{
			Client:    kubeClient,
			namespace: testCase.namespace,
		}

		err := client.EnsureDefaultNamespace(&log.FakeLogger{})

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		namespaces, _ := kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
		assert.Equal(t, len(namespaces.Items), len(testCase.expectedNamespaces), "Unexpected number of namespaces after call in testCase %s", testCase.name)
		for index, namespace := range testCase.expectedNamespaces {
			assert.Equal(t, namespaces.Items[index].ObjectMeta.Name, namespace, "Unexpected namespace at index %d in testCase %s", index, testCase.name)
		}
	}
}

type ensureGoogleCloudClusterRoleBindingTestCase struct {
	name string

	context             string
	clusterRoleBindings []string

	expectedErr bool
}

func TestEnsureGoogleCloudClusterRoleBinding(t *testing.T) {
	testCases := []ensureGoogleCloudClusterRoleBindingTestCase{
		{
			name:    "Local Kubernetes",
			context: "minikube",
		},
		{
			name:                "ClusterRoleBinding already there",
			clusterRoleBindings: []string{"devspace-user"},
		},
	}

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		for _, crb := range testCase.clusterRoleBindings {
			kubeClient.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: crb,
				},
			})
		}
		client := &client{
			Client:         kubeClient,
			currentContext: testCase.context,
		}

		err := client.EnsureGoogleCloudClusterRoleBinding(&log.FakeLogger{})

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}
	}
}

type getRunningPodsWithImageTestCase struct {
	name string

	imageNames      []string
	clientNamespace string
	namespace       string
	pods            []*k8sv1.Pod

	expectedErr  bool
	expectedPods []*k8sv1.Pod
}

func TestGetRunningPodsWithImage(t *testing.T) {
	testCases := []getRunningPodsWithImageTestCase{
		{
			name:        "No pods, Wait timeout",
			expectedErr: true,
		},
		{
			name: "No given image names, return no pods after minWait",
			pods: []*k8sv1.Pod{
				{},
			},
			expectedPods:    []*k8sv1.Pod{},
			clientNamespace: "mynamespace",
		},
		{
			name:            "Running pod with image",
			clientNamespace: "mynamespace",
			imageNames:      []string{"myimage"},
			pods: []*k8sv1.Pod{
				{
					Spec: k8sv1.PodSpec{
						Containers: []k8sv1.Container{
							{
								Image: "myimage",
							},
						},
					},
					Status: k8sv1.PodStatus{
						Reason: "Running",
					},
				},
			},
			expectedPods: []*k8sv1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "mynamespace",
					},
					Spec: k8sv1.PodSpec{
						Containers: []k8sv1.Container{
							{
								Image: "myimage",
							},
						},
					},
					Status: k8sv1.PodStatus{
						Reason: "Running",
					},
				},
			},
		},
		{
			name:       "Critical status",
			namespace:  "mynamespace",
			imageNames: []string{"myimage"},
			pods: []*k8sv1.Pod{
				{
					Spec: k8sv1.PodSpec{
						Containers: []k8sv1.Container{
							{
								Image: "myimage",
							},
						},
					},
					Status: k8sv1.PodStatus{
						Reason: "Error",
					},
				},
			},
			expectedErr: true,
		},
		{
			name:       "Unknown status",
			namespace:  "mynamespace",
			imageNames: []string{"myimage"},
			pods: []*k8sv1.Pod{
				{
					Spec: k8sv1.PodSpec{
						Containers: []k8sv1.Container{
							{
								Image: "myimage",
							},
						},
					},
					Status: k8sv1.PodStatus{
						Reason: "UnknownStatus",
					},
				},
			},
			expectedErr: true,
		},
		{
			name:       "Completed status",
			namespace:  "mynamespace",
			imageNames: []string{"myimage"},
			pods: []*k8sv1.Pod{
				{
					Spec: k8sv1.PodSpec{
						Containers: []k8sv1.Container{
							{
								Image: "myimage",
							},
						},
					},
					Status: k8sv1.PodStatus{
						Reason: "Completed",
					},
				},
			},
		},
	}

	second = time.Millisecond
	defer func() { second = time.Second }()

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		for _, pod := range testCase.pods {
			kubeClient.CoreV1().Pods("mynamespace").Create(pod)
		}
		client := &client{
			Client:    kubeClient,
			namespace: testCase.clientNamespace,
		}

		pods, err := client.GetRunningPodsWithImage(testCase.imageNames, testCase.namespace, time.Second)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		podsAsYaml, err := yaml.Marshal(pods)
		assert.NilError(t, err, "Error parsing pods to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedPods)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(podsAsYaml), string(expectedAsYaml), "Unexpected pods in testCase %s", testCase.name)
	}
}

type GetNewestPodOnceRunningTestCase struct {
	name string

	labelSelector   string
	imageSelector   []string
	clientNamespace string
	namespace       string
	pods            []*k8sv1.Pod

	expectedErr bool
	expectedPod *k8sv1.Pod
}

func TestGetNewestPodOnceRunning(t *testing.T) {
	testCases := []GetNewestPodOnceRunningTestCase{
		{
			name:        "No pods timeout",
			expectedErr: true,
		},
		{
			name:            "Get newest of two running pods with imageSelector",
			clientNamespace: "mynamespace",
			imageSelector:   []string{"selectedImage"},
			pods: []*k8sv1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "Older",
						CreationTimestamp: metav1.Time{Time: time.Unix(100, 0)},
					},
					Spec: k8sv1.PodSpec{
						Containers: []k8sv1.Container{
							{
								Image: "selectedImage",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "Newer",
						CreationTimestamp: metav1.Time{Time: time.Unix(200, 0)},
					},
					Spec: k8sv1.PodSpec{
						Containers: []k8sv1.Container{
							{
								Image: "selectedImage",
							},
						},
					},
					Status: k8sv1.PodStatus{
						Reason: "Running",
					},
				},
			},
			expectedPod: &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "Newer",
					Namespace:         "mynamespace",
					CreationTimestamp: metav1.Time{Time: time.Unix(200, 0)},
				},
				Spec: k8sv1.PodSpec{
					Containers: []k8sv1.Container{
						{
							Image: "selectedImage",
						},
					},
				},
				Status: k8sv1.PodStatus{
					Reason: "Running",
				},
			},
		},
		{
			name:            "Newest Pod is a failure",
			clientNamespace: "mynamespace",
			imageSelector:   []string{"selectedImage"},
			pods: []*k8sv1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "NewestPodIsFail",
						CreationTimestamp: metav1.Time{Time: time.Unix(100, 0)},
					},
					Spec: k8sv1.PodSpec{
						Containers: []k8sv1.Container{
							{
								Image: "selectedImage",
							},
						},
					},
					Status: k8sv1.PodStatus{
						Reason: "Error",
					},
				},
			},
			expectedErr: true,
		},
	}

	second = time.Millisecond
	defer func() { second = time.Second }()

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		for _, pod := range testCase.pods {
			kubeClient.CoreV1().Pods("mynamespace").Create(pod)
		}
		client := &client{
			Client:    kubeClient,
			namespace: testCase.clientNamespace,
		}

		pod, err := client.GetNewestPodOnceRunning(testCase.labelSelector, testCase.imageSelector, testCase.namespace, time.Second)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		podAsYaml, err := yaml.Marshal(pod)
		assert.NilError(t, err, "Error parsing pod to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedPod)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(podAsYaml), string(expectedAsYaml), "Unexpected pod in testCase %s", testCase.name)
	}
}

type getPodStatusTestCase struct {
	name string

	pod k8sv1.Pod

	expectedStatus string
}

func TestGetPodStatus(t *testing.T) {
	testCases := []getPodStatusTestCase{
		{
			name: "Deleted, reason node lost",
			pod: k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: k8sv1.PodStatus{
					Reason: "NodeLost",
				},
			},
			expectedStatus: "Unknown",
		},
		{
			name: "Deleted",
			pod: k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Status: k8sv1.PodStatus{},
			},
			expectedStatus: "Terminating",
		},
		{
			name: "Init Successfully terminated",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					InitContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Terminated: &k8sv1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Init Terminated with exitcode",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					InitContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Terminated: &k8sv1.ContainerStateTerminated{
									ExitCode: 1,
								},
							},
						},
					},
				},
			},
			expectedStatus: "Init:ExitCode:1",
		},
		{
			name: "Init Terminated with signal",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					InitContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Terminated: &k8sv1.ContainerStateTerminated{
									ExitCode: 1,
									Signal:   2,
								},
							},
						},
					},
				},
			},
			expectedStatus: "Init:Signal:2",
		},
		{
			name: "Init Terminated with reason",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					InitContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Terminated: &k8sv1.ContainerStateTerminated{
									ExitCode: 1,
									Signal:   2,
									Reason:   "someReason",
								},
							},
						},
					},
				},
			},
			expectedStatus: "Init:someReason",
		},
		{
			name: "Init Waiting with reason",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					InitContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Waiting: &k8sv1.ContainerStateWaiting{
									Reason: "someWaitReason",
								},
							},
						},
					},
				},
			},
			expectedStatus: "Init:someWaitReason",
		},
		{
			name: "pod is initializing",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					InitContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Waiting: &k8sv1.ContainerStateWaiting{
									Reason: "PodInitializing",
								},
							},
						},
					},
				},
			},
			expectedStatus: "Init:0/0",
		},
		{
			name: "Waiting reason is completed",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					ContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Waiting: &k8sv1.ContainerStateWaiting{
									Reason: "Completed",
								},
							},
						},
						{
							Ready: true,
							State: k8sv1.ContainerState{
								Running: &k8sv1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			expectedStatus: "Running",
		},
		{
			name: "Container terminated with reason",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					ContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Terminated: &k8sv1.ContainerStateTerminated{
									Reason: "terminatedReason",
								},
							},
						},
					},
				},
			},
			expectedStatus: "terminatedReason",
		},
		{
			name: "Container terminated with signal",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					ContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Terminated: &k8sv1.ContainerStateTerminated{
									Signal: 1,
								},
							},
						},
					},
				},
			},
			expectedStatus: "Signal:1",
		},
		{
			name: "Container terminated with exit code",
			pod: k8sv1.Pod{
				Status: k8sv1.PodStatus{
					ContainerStatuses: []k8sv1.ContainerStatus{
						{
							State: k8sv1.ContainerState{
								Terminated: &k8sv1.ContainerStateTerminated{
									ExitCode: 1,
								},
							},
						},
					},
				},
			},
			expectedStatus: "ExitCode:1",
		},
	}

	for _, testCase := range testCases {
		status := GetPodStatus(&testCase.pod)

		assert.Equal(t, status, testCase.expectedStatus, "Unexpected status in testCase %s", testCase.name)
	}
}
