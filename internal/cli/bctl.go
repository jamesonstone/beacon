package cli

import (
	"github.com/spf13/cobra"
)

type bctlScanOptions struct {
	noRefresh   bool
	includeIdle bool
	jsonOutput  bool
}

func (a App) BctlRoot() *cobra.Command {
	var configPath string
	var colorMode string
	var options bctlScanOptions
	root := &cobra.Command{
		Use:           "bctl",
		Short:         "Scan selected projects for work in progress",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.runBctlScan(cmd.Context(), configPath, nil, options, colorMode)
		},
	}
	root.PersistentFlags().StringVar(&configPath, "config", "", "Beacon configuration file path")
	root.PersistentFlags().StringVar(&colorMode, "color", "auto", "color output: auto, always, or never")
	addBctlScanFlags(root, &options)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	root.AddCommand(
		a.bctlScanCommand(&configPath),
		a.configuredProjectsCommand(&configPath),
		versionCommand(a.Out, "bctl"),
	)
	return root
}

func addBctlScanFlags(command *cobra.Command, options *bctlScanOptions) {
	command.Flags().BoolVar(&options.jsonOutput, "json", false, "emit JSON only")
	command.Flags().BoolVar(&options.noRefresh, "no-refresh", false, "skip git fetch")
	command.Flags().BoolVar(&options.includeIdle, "include-idle", false, "show projects with only idle work")
}
