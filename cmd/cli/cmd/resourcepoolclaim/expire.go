// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepoolclaim

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

// NewCmdExpire returns the expire subcommand for ResourcePoolClaim.
func NewCmdExpire(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	opts := &ExpireOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "expire NAME",
		Short: "Expire and release a ResourcePoolClaim",
		Example: `  # Expire a ResourcePoolClaim in current namespace
  kubectl capsule rpc expire my-claim

  # Expire a ResourcePoolClaim in a specific namespace
  kubectl capsule resource-pool-claim expire my-claim -n tenant-dev`,
		Args: cobra.ExactArgs(1),
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

	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "", "Namespace of the ResourcePoolClaim (optional, defaults to current context)")

	return cmd
}
