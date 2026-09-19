// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package namespace

import (
	"fmt"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/cordon"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/spf13/cobra"
)

// NewCmdNamespaceCordon returns the namespace cordon subcommand.
func NewCmdNamespaceCordon(f factory.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	return newCmdNamespaceCordonToggle(f, streams, true)
}

// NewCmdNamespaceUncordon returns the namespace uncordon subcommand.
func NewCmdNamespaceUncordon(f factory.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	return newCmdNamespaceCordonToggle(f, streams, false)
}

func newCmdNamespaceCordonToggle(f factory.Factory, streams genericiooptions.IOStreams, cordoned bool) *cobra.Command {
	verb := "cordon"
	short := "Cordon a Namespace to prevent workload creation and modification"
	if !cordoned {
		verb = "uncordon"
		short = "Uncordon a Namespace to resume workload operations"
	}

	opts := &cordon.CordonOptions{
		Factory:         f,
		IOStreams:       streams,
		DesiredCordoned: cordoned,
		ResourceType:    "namespace",
	}

	cmd := &cobra.Command{
		Use:   verb + " NAME",
		Short: short,
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf(`  # %s namespace 'backend-dev'
  kubectl capsule namespace %s backend-dev`, verb, verb),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ResourceName = args[0]
			if err := opts.Complete(cmd, []string{"namespace", args[0]}); err != nil {
				return err
			}
			return opts.Run(cmd.Context())
		},
	}

	return cmd
}
