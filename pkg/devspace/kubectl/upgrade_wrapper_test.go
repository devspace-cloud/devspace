package kubectl

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/devspace-cloud/devspace/pkg/devspace/kubectl/transport"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/rest"
)

type fakeUpgrader struct{}

func (u *fakeUpgrader) NewConnection(resp *http.Response) (httpstream.Connection, error) {
	return &fakeConnection{}, nil
}

type fakeConnection struct {
	Closed bool
}

func (c *fakeConnection) CreateStream(headers http.Header) (httpstream.Stream, error) {
	return nil, nil
}
func (c *fakeConnection) Close() error {
	c.Closed = true
	return nil
}
func (c *fakeConnection) CloseChan() <-chan bool {
	return make(chan bool)
}
func (c *fakeConnection) SetIdleTimeout(timeout time.Duration) {}

type newConnectionTestCase struct {
	name string

	connectionContent string

	expectedErr         bool
	expectedConnection  httpstream.Connection
	expectedConnections []httpstream.Connection
}

func TestNewConnection(t *testing.T) {
	testCases := []newConnectionTestCase{
		{
			name:                "New fake connection",
			expectedConnection:  &fakeConnection{},
			expectedConnections: []httpstream.Connection{&fakeConnection{}},
		},
	}

	for _, testCase := range testCases {
		wrapper := &upgraderWrapper{
			Upgrader: &fakeUpgrader{},
		}

		con, err := wrapper.NewConnection(&http.Response{Body: ioutil.NopCloser(bytes.NewReader([]byte(testCase.connectionContent)))})

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		conAsYaml, err := yaml.Marshal(con)
		assert.NilError(t, err, "Error parsing connection to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedConnection)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(conAsYaml), string(expectedAsYaml), "Unexpected connection in testCase %s", testCase.name)

		consAsYaml, err := yaml.Marshal(wrapper.Connections)
		assert.NilError(t, err, "Error parsing connections to yaml in testCase %s", testCase.name)
		expectedAsYaml, err = yaml.Marshal(testCase.expectedConnections)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(consAsYaml), string(expectedAsYaml), "Unexpected connections in testCase %s", testCase.name)
	}
}

type closeTestCase struct {
	name string

	connections []httpstream.Connection

	expectedErr         bool
	expectedConnections []httpstream.Connection
}

func TestClose(t *testing.T) {
	testCases := []closeTestCase{
		{
			name:                "New fake connection",
			connections:         []httpstream.Connection{&fakeConnection{}},
			expectedConnections: []httpstream.Connection{&fakeConnection{Closed: true}},
		},
	}

	for _, testCase := range testCases {
		wrapper := &upgraderWrapper{
			Upgrader:    &fakeUpgrader{},
			Connections: testCase.connections,
		}

		err := wrapper.Close()

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		consAsYaml, err := yaml.Marshal(wrapper.Connections)
		assert.NilError(t, err, "Error parsing connections to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedConnections)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(consAsYaml), string(expectedAsYaml), "Unexpected connections in testCase %s", testCase.name)
	}
}

type fakeRoundTripper struct {
	httpstream.Dialer
}

type getUpgraderWrapperTestCase struct {
	name string

	restConfig *rest.Config

	expectedErr             bool
	expectedWrapper         interface{}
	expectedUpgraderWrapper UpgraderWrapper
}

func TestGetUpgraderWrapper(t *testing.T) {
	testCases := []getUpgraderWrapperTestCase{
		{
			name:            "Get for empty rest config",
			restConfig:      &rest.Config{},
			expectedWrapper: fakeRoundTripper{},
			expectedUpgraderWrapper: &upgraderWrapper{
				Upgrader: transport.NewRoundTripper(nil, true, false),
			},
		},
	}

	for _, testCase := range testCases {
		client := &client{
			restConfig: testCase.restConfig,
		}

		wrapper, upgraderWrapper, err := client.GetUpgraderWrapper()

		if !testCase.expectedErr {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else if err == nil {
			t.Fatalf("Unexpected no error in testCase %s", testCase.name)
		}

		wrapperAsYaml, err := yaml.Marshal(wrapper)
		assert.NilError(t, err, "Error parsing wrapper to yaml in testCase %s", testCase.name)
		expectedAsYaml, err := yaml.Marshal(testCase.expectedWrapper)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(wrapperAsYaml), string(expectedAsYaml), "Unexpected wrapper in testCase %s", testCase.name)

		wrapperAsYaml, err = yaml.Marshal(upgraderWrapper)
		assert.NilError(t, err, "Error parsing pugrader wrapper to yaml in testCase %s", testCase.name)
		expectedAsYaml, err = yaml.Marshal(testCase.expectedUpgraderWrapper)
		assert.NilError(t, err, "Error parsing expection to yaml in testCase %s", testCase.name)
		assert.Equal(t, string(wrapperAsYaml), string(expectedAsYaml), "Unexpected upgrader wrapper in testCase %s", testCase.name)
	}
}
