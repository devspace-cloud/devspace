package build

import (
	"sort"
	"testing"

	fakebuilder "github.com/devspace-cloud/devspace/pkg/devspace/build/builder/testing"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/generated"
	"github.com/devspace-cloud/devspace/pkg/devspace/config/versions/latest"
	fakehook "github.com/devspace-cloud/devspace/pkg/devspace/hook/testing"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

type buildTestCase struct {
	name string

	options Options
	cache   *generated.CacheConfig
	images  map[string]*latest.ImageConfig

	expectedErr         string
	expectedBuiltImages map[string]string
	expectedCache       *generated.CacheConfig
}

func TestBuild(t *testing.T) {
	testCases := []buildTestCase{
		buildTestCase{
			name: "No images to build",
		},
		buildTestCase{
			name: "Skip build",
			images: map[string]*latest.ImageConfig{
				"myImage": &latest.ImageConfig{
					Build: &latest.BuildConfig{
						Custom: &latest.CustomConfig{},
					},
				},
			},
		},
		buildTestCase{
			name: "One sequencial build",
			images: map[string]*latest.ImageConfig{
				"myImage": &latest.ImageConfig{
					Image: "myImage",
				},
			},
			options: Options{
				ForceRebuild: true,
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{},
			},
			expectedBuiltImages: map[string]string{
				"myImage": "",
			},
			expectedCache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"myImage": &generated.ImageCache{
						ImageName: "myImage",
					},
				},
			},
		},
		buildTestCase{
			name: "TWo non-sequencial builds",
			images: map[string]*latest.ImageConfig{
				"image1": &latest.ImageConfig{
					Image: "firstimage",
				},
				"image2": &latest.ImageConfig{
					Image: "secoundimage",
				},
			},
			options: Options{
				ForceRebuild: true,
			},
			cache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{},
			},
			expectedBuiltImages: map[string]string{
				"firstimage":   "",
				"secoundimage": "",
			},
			expectedCache: &generated.CacheConfig{
				Images: map[string]*generated.ImageCache{
					"image1": &generated.ImageCache{
						ImageName: "firstimage",
					},
					"image2": &generated.ImageCache{
						ImageName: "secoundimage",
					},
				},
			},
		},
	}

	defer func() { overwriteBuilder = nil }()

	for _, testCase := range testCases {
		controller := &controller{
			config: &latest.Config{
				Images: testCase.images,
			},
			cache:        testCase.cache,
			hookExecuter: &fakehook.FakeHook{},
		}
		overwriteBuilder = &fakebuilder.Builder{}

		builtImages, err := controller.Build(&testCase.options, log.Discard)

		if testCase.expectedErr == "" {
			assert.NilError(t, err, "Error in testCase %s", testCase.name)
		} else {
			assert.Error(t, err, testCase.expectedErr, "Wrong or no error in testCase %s", testCase.name)
		}

		builtImagesKeys := getKeys(builtImages)
		expectationKeys := getKeys(testCase.expectedBuiltImages)
		assert.Equal(t, string(builtImagesKeys), string(expectationKeys), "Unexpected builtImages in testCase %s", testCase.name)

		isCacheEqual(t, testCase.cache, testCase.expectedCache, testCase.name)
	}
}

func getKeys(targetMap map[string]string) string {
	arr := []string{}
	for key := range targetMap {
		arr = append(arr, key)
	}
	sort.Strings(arr)

	result := ""
	for _, key := range arr {
		result += key + ", "
	}
	return result
}

func isCacheEqual(t *testing.T, cache1 *generated.CacheConfig, cache2 *generated.CacheConfig, testCase string) {
	if cache1 != nil && cache2 != nil && cache1.Images != nil && cache2.Images != nil {
		for key, imageConfig := range cache2.Images {
			if cache1ImageConfig, ok := cache1.Images[key]; ok {
				imageConfig.Tag = cache1ImageConfig.Tag
			}
		}
	}

	cache1AsYaml, err := yaml.Marshal(cache1)
	assert.NilError(t, err, "Error marshaling cache in testCase %s", testCase)
	cache2AsYaml, err := yaml.Marshal(cache2)
	assert.NilError(t, err, "Error marshaling expected cache in testCase %s", testCase)
	assert.Equal(t, string(cache1AsYaml), string(cache2AsYaml), "Unexpected cache in testCase %s", testCase)

}
