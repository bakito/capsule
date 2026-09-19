// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepoolclaim

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

// NewCmdResourcePoolClaim returns the resource-pool-claim parent command.
func NewCmdResourcePoolClaim(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resource-pool-claim",
		Aliases: []string{"resourcepoolclaim", "rpc", "claim"},
		Short:   "Manage ResourcePoolClaims",
		Long:    "Manage lifecycle operations on Capsule ResourcePoolClaims.",
	}

	cmd.AddCommand(
		NewCmdExpire(f, streams),
	)

	return cmd
}
