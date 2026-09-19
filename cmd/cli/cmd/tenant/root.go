// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/spf13/cobra"
)

// NewCmdTenant returns the tenant parent command.
func NewCmdTenant(f factory.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tenant",
		Aliases: []string{"tenants", "tnt"},
		Short:   "Manage Capsule Tenants",
		Long:    "Create, get, describe, cordon, and uncordon Capsule Tenants.",
	}

	cmd.AddCommand(
		NewCmdGet(f, streams),
		NewCmdDescribe(f, streams),
		NewCmdCreate(f, streams),
		NewCmdTenantCordon(f, streams),
		NewCmdTenantUncordon(f, streams),
	)

	return cmd
}
