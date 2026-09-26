package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/open-nerve/NerveProject/server/internal/bootstrap"
)

// newUsersCommand is `nerve users`, the server administrator's commands
// (M2 design 3.17). Each names the account with --email; the ones that set
// a password read it from stdin.
func newUsersCommand(load configLoader, stdin io.Reader) *cobra.Command {
	users := &cobra.Command{
		Use:   "users",
		Short: "Manage accounts as the server's administrator",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	users.AddCommand(
		userCommand(load, stdin, "create", "Create an account without signing it in; works while sign-up is closed",
			true, bootstrap.CreateUser),
		userCommand(load, stdin, "reset-password", "Set an account's password and revoke all its sessions and API tokens",
			true, bootstrap.ResetPassword),
		setEmailCommand(load),
		userCommand(load, stdin, "deactivate", "Deactivate an account and revoke its sessions; its API tokens stay",
			false, func(email, _ string) bootstrap.UserCommand { return bootstrap.DeactivateUser(email) }),
		userCommand(load, stdin, "activate", "Activate an account; its unexpired API tokens work again, so run reset-password too if it may be compromised",
			false, func(email, _ string) bootstrap.UserCommand { return bootstrap.ActivateUser(email) }),
	)
	return users
}

// userCommand builds one `nerve users` subcommand: it requires --email and,
// when withPassword, reads the password before it runs.
func userCommand(load configLoader, stdin io.Reader, use, short string, withPassword bool,
	command func(email, password string) bootstrap.UserCommand) *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:   use + " --email <address>",
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			var password string
			if withPassword {
				if password, err = readPassword(stdin, cmd.ErrOrStderr()); err != nil {
					return err
				}
			}
			return bootstrap.Users(cmd.Context(), cfg, cmd.ErrOrStderr(), cmd.OutOrStdout(), command(email, password))
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "the account's e-mail address")
	_ = cmd.MarkFlagRequired("email") // the flag exists
	return cmd
}

// setEmailCommand is `nerve users set-email` (M2 decision 1).
func setEmailCommand(load configLoader) *cobra.Command {
	var email, newEmail string
	cmd := &cobra.Command{
		Use: "set-email --email <address> --new-email <address>",
		Short: "Change an account's e-mail address and revoke its sessions; its API tokens stay, " +
			"so run reset-password too if the change is about a compromise",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return bootstrap.Users(cmd.Context(), cfg, cmd.ErrOrStderr(), cmd.OutOrStdout(), bootstrap.SetEmail(email, newEmail))
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "the account's e-mail address")
	cmd.Flags().StringVar(&newEmail, "new-email", "", "the new e-mail address")
	_ = cmd.MarkFlagRequired("email") // the flags exist
	_ = cmd.MarkFlagRequired("new-email")
	return cmd
}

// readPassword reads the new password (M2 design 3.17): on a terminal it
// asks twice without echo, and the two must match; otherwise it reads one
// line, for scripts and tests. Only the line ending is removed.
func readPassword(in io.Reader, prompt io.Writer) (string, error) {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		first, err := promptPassword(f, prompt, "Password: ")
		if err != nil {
			return "", err
		}
		again, err := promptPassword(f, prompt, "Password again: ")
		if err != nil {
			return "", err
		}
		if !bytes.Equal(first, again) {
			return "", errors.New("the passwords do not match")
		}
		return string(first), nil
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", fmt.Errorf("read the password from standard input: %w", err)
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func promptPassword(f *os.File, prompt io.Writer, label string) ([]byte, error) {
	_, _ = io.WriteString(prompt, label)
	password, err := term.ReadPassword(int(f.Fd()))
	_, _ = io.WriteString(prompt, "\n")
	if err != nil {
		return nil, fmt.Errorf("read the password: %w", err)
	}
	return password, nil
}
