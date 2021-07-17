package variable

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/kubectl/util"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"github.com/loft-sh/devspace/pkg/devspace/upgrade"
	"github.com/loft-sh/devspace/pkg/util/git"
	"github.com/loft-sh/devspace/pkg/util/kubeconfig"
	"github.com/loft-sh/devspace/pkg/util/randutil"
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
    "regexp"
)

// PredefinedVariableOptions holds the options for a predefined variable to load
type PredefinedVariableOptions struct {
	BasePath   string
	ConfigPath string

	KubeContextFlag  string
	NamespaceFlag    string
	KubeConfigLoader kubeconfig.Loader

	Profile string
}

// PredefinedVariableFunction is the definition of a predefined variable
type PredefinedVariableFunction func(options *PredefinedVariableOptions) (interface{}, error)

// predefinedVars holds all predefined variables that can be used in the config
var predefinedVars = map[string]PredefinedVariableFunction{
	"DEVSPACE_VERSION": func(options *PredefinedVariableOptions) (interface{}, error) {
		return upgrade.GetVersion(), nil
	},
	"DEVSPACE_RANDOM": func(options *PredefinedVariableOptions) (interface{}, error) {
		return randutil.GenerateRandomString(6), nil
	},
	"DEVSPACE_PROFILE": func(options *PredefinedVariableOptions) (interface{}, error) {
		return options.Profile, nil
	},
	"DEVSPACE_USER_HOME": func(options *PredefinedVariableOptions) (interface{}, error) {
		homeDir, err := homedir.Dir()
		if err != nil {
			return nil, err
		}
		return homeDir, nil
	},
	"DEVSPACE_TIMESTAMP": func(options *PredefinedVariableOptions) (interface{}, error) {
		return strconv.FormatInt(time.Now().Unix(), 10), nil
	},
	"DEVSPACE_GIT_BRANCH": func(options *PredefinedVariableOptions) (interface{}, error) {
		configPath := options.BasePath
		if configPath == "" {
			configPath = options.ConfigPath
		}

		branch, err := git.GetBranch(filepath.Dir(configPath))
		if err != nil {
			return "", fmt.Errorf("error retrieving git branch: %v, but predefined var DEVSPACE_GIT_BRANCH is used", err)
		}

		return branch, nil
	},
	"DEVSPACE_GIT_BRANCH_SLUG": func(options *PredefinedVariableOptions) (interface{}, error) {
		configPath := options.BasePath
		if configPath == "" {
			configPath = options.ConfigPath
		}

		branch, err := git.GetBranch(filepath.Dir(configPath))
		if err != nil {
			return "", fmt.Errorf("error retrieving git branch: %v, but predefined var DEVSPACE_GIT_BRANCH_SLUG is used", err)
		}
        reg, err := regexp.Compile("[^a-zA-Z0-9]+")
        branchSlug := reg.ReplaceAllString(branch, "-")
		if err != nil {
			return "", fmt.Errorf("error slug regexp for branch: %v, but predefined var DEVSPACE_GIT_BRANCH_SLUG is used", err)
		}

		return branchSlug, nil
	},
	"DEVSPACE_GIT_COMMIT": func(options *PredefinedVariableOptions) (interface{}, error) {
		configPath := options.BasePath
		if configPath == "" {
			configPath = options.ConfigPath
		}

		hash, err := git.GetHash(filepath.Dir(configPath))
		if err != nil {
			return "", fmt.Errorf("no git repository found (%v), but predefined var DEVSPACE_GIT_COMMIT is used", err)
		}

		return hash[:8], nil
	},
	"DEVSPACE_CONTEXT": func(options *PredefinedVariableOptions) (interface{}, error) {
		_, activeContext, _, _, err := util.NewClientByContext(options.KubeContextFlag, options.NamespaceFlag, false, options.KubeConfigLoader)
		if err != nil {
			return "", err
		}

		return activeContext, nil
	},
	"DEVSPACE_NAMESPACE": func(options *PredefinedVariableOptions) (interface{}, error) {
		_, _, activeNamespace, _, err := util.NewClientByContext(options.KubeContextFlag, options.NamespaceFlag, false, options.KubeConfigLoader)
		if err != nil {
			return "", err
		}

		return activeNamespace, nil
	},
}

func IsPredefinedVariable(name string) bool {
	name = strings.ToUpper(name)
	_, ok := predefinedVars[name]
	return ok
}

func AddPredefinedVars(plugins []plugin.Metadata) {
	for _, p := range plugins {
		pluginName := p.Name
		pluginFolder := p.PluginFolder
		for _, variable := range p.Vars {
			v := variable
			predefinedVars[variable.Name] = func(options *PredefinedVariableOptions) (interface{}, error) {
				args, err := json.Marshal(os.Args)
				if err != nil {
					return "", err
				}

				buffer := &bytes.Buffer{}
				err = plugin.CallPluginExecutable(filepath.Join(pluginFolder, plugin.PluginBinary), v.BaseArgs, map[string]string{
					plugin.KubeContextFlagEnv:   options.KubeContextFlag,
					plugin.KubeNamespaceFlagEnv: options.NamespaceFlag,
					plugin.OsArgsEnv:            string(args),
				}, buffer)
				if err != nil {
					return "", fmt.Errorf("executing plugin %s: %s - %v", pluginName, buffer.String(), err)
				}

				return strings.TrimSpace(buffer.String()), nil
			}
		}
	}
}

// NewPredefinedVariable creates a new predefined variable for the given name or fails if there
// is none with the given name
func NewPredefinedVariable(name string, cache map[string]string, options *PredefinedVariableOptions) (Variable, error) {
	name = strings.ToUpper(name)
	if _, ok := predefinedVars[name]; !ok {
		// Load space domain environment variable
		if strings.HasPrefix(name, "DEVSPACE_SPACE_DOMAIN") {
			// Check if its in generated config
			if val, ok := cache[name]; ok {
				return NewCachedValueVariable(val), nil
			}

			return NewCachedValueVariable(name), nil
		}

		return nil, errors.New("predefined variable " + name + " not found")
	}

	return &predefinedVariable{
		name:    name,
		cache:   cache,
		options: options,
	}, nil
}

type predefinedVariable struct {
	name    string
	cache   map[string]string
	options *PredefinedVariableOptions
}

func (p *predefinedVariable) Load(definition *latest.Variable) (interface{}, error) {
	name := strings.ToUpper(p.name)
	getVar, ok := predefinedVars[name]
	if !ok {
		return nil, errors.New("predefined variable " + name + " not found")
	}

	return getVar(p.options)
}
