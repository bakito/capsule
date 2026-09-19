// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/cordon"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/expire"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/namespace"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/reconcile"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/resourcepermit"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/resourcepoolclaim"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/tenant"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/version"
	capsuleversion "github.com/projectcapsule/capsule/internal/version"
)

// NewRootCmd creates the root command for the Capsule CLI plugin.
func NewRootCmd(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "capsule",
		Short:        "kubectl plugin for Capsule multi-tenancy",
		Long:         "A CLI and kubectl plugin for managing Capsule Tenants, Namespaces, ResourcePermits, and Resources.",
		Version:      fmt.Sprintf("%s %s%s", capsuleversion.GitTag, capsuleversion.GitCommit, capsuleversion.GitDirty),
		SilenceUsage: true,
	}

	if f != nil && f.ConfigFlags() != nil {
		f.ConfigFlags().AddFlags(cmd.PersistentFlags())
	}

	// Subcommands
	cmd.AddCommand(
		tenant.NewCmdTenant(f, streams),
		namespace.NewCmdNamespace(f, streams),
		cordon.NewCmdCordon(f, streams),
		cordon.NewCmdUncordon(f, streams),
		reconcile.NewCmdReconcile(f, streams),
		expire.NewCmdExpire(f, streams),
		resourcepoolclaim.NewCmdResourcePoolClaim(f, streams),
		resourcepermit.NewCmdResourcePermit(f, streams),
		version.NewCmdVersion(streams),
	)

	return cmd
}

// Execute is the main entry point called from main.go.
func Execute() {
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	f := factory.NewFactory(configFlags)
	streams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	rootCmd := NewRootCmd(f, streams)
	if err := rootCmd.Execute(); err != nil {
		//nolint:forbidigo // standard CLI error exit
		fmt.Println(err)
		os.Exit(1)
	}
}
