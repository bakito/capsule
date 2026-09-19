// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

// NewCmdTenant returns the tenant parent command.
func NewCmdTenant(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tenant",
		Aliases: []string{"tenants", "tnt"},
		Short:   "Manage Capsule Tenants",
		Long:    "Create, describe, cordon, and uncordon Capsule Tenants.",
	}

	cmd.AddCommand(
		NewCmdCreate(f, streams),
		NewCmdDescribe(f, streams),
		NewCmdTenantCordon(f, streams),
		NewCmdTenantUncordon(f, streams),
	)

	return cmd
}
