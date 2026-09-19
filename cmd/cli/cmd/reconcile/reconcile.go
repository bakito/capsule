// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package reconcile

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

// NewCmdReconcile returns the top-level reconcile command.
func NewCmdReconcile(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	opts := &ReconcileOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "reconcile (TYPE NAME | TYPE/NAME)",
		Short: "Request on-demand reconciliation of a GlobalTenantResource or TenantResource",
		Example: `  # Trigger reconcile for a GlobalTenantResource
  kubectl capsule reconcile gtr default-policies

  # Trigger reconcile for a TenantResource in a namespace
  kubectl capsule reconcile tr local-rules -n oil-dev

  # Trigger reconcile using type/name format
  kubectl capsule reconcile globaltenantresource/default-policies`,
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

	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "", "Namespace of the TenantResource (optional, defaults to current context)")

	return cmd
}
