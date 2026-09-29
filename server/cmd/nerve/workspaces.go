package main

import (
	"github.com/spf13/cobra"

	"github.com/open-nerve/NerveProject/server/internal/bootstrap"
)

// newWorkspacesCommand is `nerve workspaces`, the server administrator's
// commands on workspaces (M3 design 3.11).
func newWorkspacesCommand(load configLoader) *cobra.Command {
	workspaces := &cobra.Command{
		Use:   "workspaces",
		Short: "Manage workspaces as the server's administrator",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	workspaces.AddCommand(createWorkspaceCommand(load))
	return workspaces
}

// createWorkspaceCommand is `nerve workspaces create`: it works while
// workspace creation is switched off.
func createWorkspaceCommand(load configLoader) *cobra.Command {
	var slug, name, adminEmail string
	cmd := &cobra.Command{
		Use:   "create --slug <slug> --name <name> --admin-email <address>",
		Short: "Create a workspace with an account as its admin; works while workspace creation is switched off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return bootstrap.Workspaces(cmd.Context(), cfg, cmd.ErrOrStderr(), cmd.OutOrStdout(),
				bootstrap.CreateWorkspace(slug, name, adminEmail))
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "the workspace's slug, its address")
	cmd.Flags().StringVar(&name, "name", "", "the workspace's name")
	cmd.Flags().StringVar(&adminEmail, "admin-email", "", "the e-mail address of the account that becomes its admin")
	_ = cmd.MarkFlagRequired("slug") // the flags exist
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("admin-email")
	return cmd
}
