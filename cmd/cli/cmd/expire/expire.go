// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package expire

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

// NewCmdExpire returns the top-level expire command.
func NewCmdExpire(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	opts := &ExpireOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "expire (TYPE NAME | TYPE/NAME)",
		Short: "Expire a ResourcePoolClaim or ResourcePermit",
		Example: `  # Expire a ResourcePoolClaim
  kubectl capsule expire resourcepoolclaim my-claim -n dev-tenant
  kubectl capsule expire rpc my-claim -n dev-tenant

  # Expire a ResourcePermit
  kubectl capsule expire resourcepermit grant-admin -n default
  kubectl capsule expire rp/grant-admin`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(cmd, args); err != nil {
				return err
			}

			if err := opts.Validate(); err != nil {
				return err
			}

			return opts.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "", "Namespace of the resource (optional, defaults to current context)")

	return cmd
}
