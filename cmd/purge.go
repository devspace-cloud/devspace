package cmd

import (
	"github.com/loft-sh/devspace/pkg/devspace/dependency/types"
	"github.com/loft-sh/devspace/pkg/devspace/plugin"
	"strings"

	"github.com/loft-sh/devspace/pkg/util/factory"

	"github.com/loft-sh/devspace/cmd/flags"
	"github.com/loft-sh/devspace/pkg/devspace/dependency"
	"github.com/loft-sh/devspace/pkg/util/log"
	"github.com/loft-sh/devspace/pkg/util/message"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// PurgeCmd holds the required data for the purge cmd
type PurgeCmd struct {
	*flags.GlobalFlags

	Deployments         string
	VerboseDependencies bool
	PurgeDependencies   bool
	All                 bool

	Dependency []string

	log log.Logger
}

// NewPurgeCmd creates a new purge command
func NewPurgeCmd(f factory.Factory, globalFlags *flags.GlobalFlags, plugins []plugin.Metadata) *cobra.Command {
	cmd := &PurgeCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	purgeCmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete deployed resources",
		Long: `
#######################################################
################### devspace purge ####################
#######################################################
Deletes the deployed kuberenetes resources:

devspace purge
devspace purge --dependencies
devspace purge -d my-deployment
#######################################################`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(f, plugins, cobraCmd, args)
		},
	}

	purgeCmd.Flags().StringVarP(&cmd.Deployments, "deployments", "d", "", "The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)")
	purgeCmd.Flags().BoolVarP(&cmd.All, "all", "a", false, "When enabled purges the dependencies as well")
	purgeCmd.Flags().BoolVar(&cmd.PurgeDependencies, "dependencies", false, "DEPRECATED: Please use --all instead")
	purgeCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", true, "Builds the dependencies verbosely")

	purgeCmd.Flags().StringSliceVar(&cmd.Dependency, "dependency", []string{}, "Purges only the specific named dependencies")

	return purgeCmd
}

// Run executes the purge command logic
func (cmd *PurgeCmd) Run(f factory.Factory, plugins []plugin.Metadata, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	cmd.log = f.GetLog()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(cmd.ConfigPath)
	configExists, err := configLoader.SetDevSpaceRoot(cmd.log)
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	// check for deprecated flag
	if cmd.PurgeDependencies {
		cmd.log.Warnf("Flag --dependencies is deprecated, please use --all or -a instead")
		cmd.All = true
	}

	log.StartFileLogging()

	// Get config with adjusted cluster config
	generatedConfig, err := configLoader.LoadGenerated(configOptions)
	if err != nil {
		return err
	}
	configOptions.GeneratedConfig = generatedConfig

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}
	configOptions.KubeClient = client

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, cmd.log)
	if err != nil {
		return err
	}

	// Execute plugin hook
	err = plugin.ExecutePluginHook(plugins, cobraCmd, args, "purge", client.CurrentContext(), client.Namespace(), nil)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	configInterface, err := configLoader.Load(configOptions, cmd.log)
	if err != nil {
		return err
	}

	// Purge dependencies
	var dependencies []types.Dependency
	if cmd.All || len(cmd.Dependency) > 0 {
		dependencies, err = f.NewDependencyManager(configInterface, client, cmd.ToConfigOptions(), cmd.log).PurgeAll(dependency.PurgeOptions{
			Dependencies: cmd.Dependency,
			Verbose:      cmd.VerboseDependencies,
		})
		if err != nil {
			cmd.log.Errorf("Error purging dependencies: %v", err)
		}
	}

	// Only purge if we don't specify dependency
	if len(cmd.Dependency) == 0 {
		deployments := []string{}
		if cmd.Deployments != "" {
			deployments = strings.Split(cmd.Deployments, ",")
			for index := range deployments {
				deployments[index] = strings.TrimSpace(deployments[index])
			}
		}

		// Purge deployments
		err = f.NewDeployController(configInterface, dependencies, client).Purge(deployments, cmd.log)
		if err != nil {
			cmd.log.Errorf("Error purging deployments: %v", err)
		}
	}

	err = configLoader.SaveGenerated(generatedConfig)
	if err != nil {
		cmd.log.Errorf("Error saving generated.yaml: %v", err)
	}

	return nil
}
