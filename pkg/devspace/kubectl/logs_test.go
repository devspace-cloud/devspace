package kubectl

type readLogsTestCase struct {
	name string

	namespace        string
	podName          string
	containerName    string
	lastContainerLog bool
	tail             *int64

	expectedErr  bool
	expectedLogs string
}

/*func TestReadLogs(t *testing.T) {
	testCases := []readLogsTestCase{
		{
			name:        "No Request URL",
			expectedErr: true,
		},
	}

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		kubeClient.CoreV1().Pods("nsWithPods").Create(&k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "somePod",
			},
			Status: k8sv1.PodStatus{},
		})
		client := &client{
			Client: kubeClient,
		}

		logs, err := client.ReadLogs(testCase.namespace, testCase.podName, testCase.containerName, testCase.lastContainerLog, testCase.tail)

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}
		assert.Equal(t, logs, testCase.expectedLogs, "Unexpected logs in testCase %s", testCase.name)
	}
}*/

type logMultipleTestCase struct {
	name string

	imageSelector []string
	tail          *int64

	expectedErr  bool
	expectedLogs string
}

/*func TestLogMultiple(t *testing.T) {
	testCases := []logMultipleTestCase{
		{
			imageSelector: []string{""},
		},
	}

	for _, testCase := range testCases {
		kubeClient := fake.NewSimpleClientset()
		kubeClient.CoreV1().Pods("nsWithPods").Create(&k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "somePod",
			},
			Spec: k8sv1.PodSpec{
				Containers: []k8sv1.Container{
					{},
				},
			},
			Status: k8sv1.PodStatus{
				Reason: "Running",
			},
		})
		client := &client{
			Client: kubeClient,
		}

		reader, writer := io.Pipe()
		defer reader.Close()
		defer writer.Close()

		interrupt := make(chan error)
		defer close(interrupt)

		err := client.LogMultiple(testCase.imageSelector, interrupt, testCase.tail, writer, &log.FakeLogger{})

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}
		writer.Close()
		logs, err := ioutil.ReadAll(reader)
		assert.Equal(t, string(logs), testCase.expectedLogs, "Unexpected logs in testCase %s", testCase.name)
	}
}
*/
