// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package namespace

import (
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/spf13/cobra"
)

// NewCmdNamespace returns the namespace parent command.
func NewCmdNamespace(f factory.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "namespace",
		Aliases: []string{"namespaces", "ns"},
		Short:   "Manage Tenant membership and status for Namespaces",
		Long:    "Add, move, remove, cordon, and uncordon Namespaces in Capsule Tenants.",
	}

	cmd.AddCommand(
		NewCmdAdd(f, streams),
		NewCmdMove(f, streams),
		NewCmdRemove(f, streams),
		NewCmdNamespaceCordon(f, streams),
		NewCmdNamespaceUncordon(f, streams),
	)

	return cmd
}
