package kubectl

import (
	"testing"

	"gotest.tools/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type groupVersionExistsTestCase struct {
	name string

	groupVersion string
	resourceList []*metav1.APIResourceList

	expectedExists bool
}

func TestGroupVersionExists(t *testing.T) {
	testCases := []groupVersionExistsTestCase{
		{
			name: "Exists",
			resourceList: []*metav1.APIResourceList{
				&metav1.APIResourceList{
					GroupVersion: "notNeedle",
				},
				&metav1.APIResourceList{
					GroupVersion: "needle",
				},
			},
			groupVersion:   "needle",
			expectedExists: true,
		},
		{
			name: "Not exists",
			resourceList: []*metav1.APIResourceList{
				&metav1.APIResourceList{
					GroupVersion: "notNeedle",
				},
			},
			groupVersion: "needle",
		},
	}

	for _, testCase := range testCases {
		exists := GroupVersionExist(testCase.groupVersion, testCase.resourceList)
		assert.Equal(t, exists, testCase.expectedExists, "Unexpected result in testCase %s", testCase.name)
	}
}

type resourceExistsTestCase struct {
	name string

	groupVersion string
	nameParam    string
	resourceList []*metav1.APIResourceList

	expectedExists bool
}

func TestResourceExists(t *testing.T) {
	testCases := []resourceExistsTestCase{
		{
			name: "Exists",
			resourceList: []*metav1.APIResourceList{
				&metav1.APIResourceList{
					GroupVersion: "groupNeedle",
					APIResources: []metav1.APIResource{
						metav1.APIResource{
							Name: "nameNeedle",
						},
						metav1.APIResource{
							Name: "notNeedle",
						},
					},
				},
				&metav1.APIResourceList{
					GroupVersion: "needle",
				},
			},
			groupVersion:   "groupNeedle",
			nameParam:      "nameNeedle",
			expectedExists: true,
		},
		{
			name: "Not exists",
			resourceList: []*metav1.APIResourceList{
				&metav1.APIResourceList{
					GroupVersion: "groupNeedle",
					APIResources: []metav1.APIResource{
						metav1.APIResource{
							Name: "notNeedle",
						},
					},
				},
				&metav1.APIResourceList{
					GroupVersion: "notNeedle",
					APIResources: []metav1.APIResource{
						metav1.APIResource{
							Name: "nameNeedle",
						},
					},
				},
			},
			groupVersion: "groupNeedle",
			nameParam:    "nameNeedle",
		},
	}

	for _, testCase := range testCases {
		exists := ResourceExist(testCase.groupVersion, testCase.nameParam, testCase.resourceList)
		assert.Equal(t, exists, testCase.expectedExists, "Unexpected result in testCase %s", testCase.name)
	}
}
