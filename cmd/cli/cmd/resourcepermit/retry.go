// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepermit

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

// NewCmdRetry returns the retry subcommand.
func NewCmdRetry(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := &ActionOptions{
		Factory:   f,
		IOStreams: streams,
		Phase:     capsulev1beta2.ResourcePermitPhaseRetrying,
	}

	cmd := &cobra.Command{
		Use:   "retry NAME [flags]",
		Short: "Retry a failed ResourcePermit",
		Args:  cobra.ExactArgs(1),
		Example: `  # Retry a failed ResourcePermit after fixing its execution identity or permissions
  kubectl capsule resource-permit retry grant-admin --namespace default`,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "Namespace of the ResourcePermit")

	return cmd
}
