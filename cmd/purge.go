package cmd

import (
	"strings"

	"github.com/devspace-cloud/devspace/pkg/util/factory"

	"github.com/devspace-cloud/devspace/cmd/flags"
	"github.com/devspace-cloud/devspace/pkg/util/log"
	"github.com/devspace-cloud/devspace/pkg/util/message"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

// PurgeCmd holds the required data for the purge cmd
type PurgeCmd struct {
	*flags.GlobalFlags

	Deployments             string
	AllowCyclicDependencies bool
	VerboseDependencies     bool
	PurgeDependencies       bool

	log log.Logger
}

// NewPurgeCmd creates a new purge command
func NewPurgeCmd(f factory.Factory, globalFlags *flags.GlobalFlags) *cobra.Command {
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
			return cmd.Run(f, cobraCmd, args)
		},
	}

	purgeCmd.Flags().StringVarP(&cmd.Deployments, "deployments", "d", "", "The deployment to delete (You can specify multiple deployments comma-separated, e.g. devspace-default,devspace-database etc.)")
	purgeCmd.Flags().BoolVar(&cmd.AllowCyclicDependencies, "allow-cyclic", false, "When enabled allows cyclic dependencies")
	purgeCmd.Flags().BoolVar(&cmd.PurgeDependencies, "dependencies", false, "When enabled purges the dependencies as well")
	purgeCmd.Flags().BoolVar(&cmd.VerboseDependencies, "verbose-dependencies", false, "Builds the dependencies verbosely")

	return purgeCmd
}

// Run executes the purge command logic
func (cmd *PurgeCmd) Run(f factory.Factory, cobraCmd *cobra.Command, args []string) error {
	// Set config root
	cmd.log = f.GetLog()
	kubeLoader := f.NewKubeConfigLoader()
	configOptions := cmd.ToConfigOptions()
	configLoader := f.NewConfigLoader(configOptions, cmd.log)
	configExists, err := configLoader.SetDevSpaceRoot()
	if err != nil {
		return err
	}
	if !configExists {
		return errors.New(message.ConfigNotFound)
	}

	log.StartFileLogging()

	// Get config with adjusted cluster config
	generatedConfig, err := configLoader.Generated()
	if err != nil {
		return err
	}

	// Use last context if specified
	err = cmd.UseLastContext(generatedConfig, cmd.log)
	if err != nil {
		return err
	}

	client, err := f.NewKubeClientFromContext(cmd.KubeContext, cmd.Namespace, cmd.SwitchContext)
	if err != nil {
		return errors.Wrap(err, "create kube client")
	}

	err = client.PrintWarning(generatedConfig, cmd.NoWarn, false, cmd.log)
	if err != nil {
		return err
	}

	// Signal that we are working on the space if there is any
	err = f.NewSpaceResumer(kubeLoader, client, cmd.log).ResumeSpace(true)
	if err != nil {
		return err
	}

	// Get config with adjusted cluster config
	config, err := configLoader.Load()
	if err != nil {
		return err
	}

	deployments := []string{}
	if cmd.Deployments != "" {
		deployments = strings.Split(cmd.Deployments, ",")
		for index := range deployments {
			deployments[index] = strings.TrimSpace(deployments[index])
		}
	}

	// Purge deployments
	err = f.NewDeployController(config, generatedConfig.GetActive(), client).Purge(deployments, cmd.log)
	if err != nil {
		cmd.log.Errorf("Error purging deployments: %v", err)
	}

	// Purge dependencies
	if cmd.PurgeDependencies {

		// Create Dependencymanager
		manager, err := f.NewDependencyManager(config, generatedConfig, kubeLoader, client, cmd.AllowCyclicDependencies, cmd.ToConfigOptions(), cmd.log)
		if err != nil {
			return errors.Wrap(err, "new manager")
		}

		err = manager.PurgeAll(cmd.VerboseDependencies)
		if err != nil {
			cmd.log.Errorf("Error purging dependencies: %v", err)
		}
	}

	err = configLoader.SaveGenerated(generatedConfig)
	if err != nil {
		cmd.log.Errorf("Error saving generated.yaml: %v", err)
	}

	return nil
}
