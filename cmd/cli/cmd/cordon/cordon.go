// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package cordon

import (
	"fmt"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/spf13/cobra"
)

// NewCmdCordon returns the top-level cordon command.
func NewCmdCordon(f factory.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	return newCordonCommand(f, streams, true)
}

// NewCmdUncordon returns the top-level uncordon command.
func NewCmdUncordon(f factory.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	return newCordonCommand(f, streams, false)
}

func newCordonCommand(f factory.Factory, streams genericiooptions.IOStreams, cordoned bool) *cobra.Command {
	verb := "cordon"
	short := "Cordon a Capsule Tenant, Namespace, GlobalTenantResource, or TenantResource"
	if !cordoned {
		verb = "uncordon"
		short = "Uncordon a Capsule Tenant, Namespace, GlobalTenantResource, or TenantResource"
	}

	opts := &CordonOptions{
		Factory:         f,
		IOStreams:       streams,
		DesiredCordoned: cordoned,
	}

	cmd := &cobra.Command{
		Use:   verb + " (TYPE NAME | TYPE/NAME)",
		Short: short,
		Example: fmt.Sprintf(`  # Cordon a Tenant
  kubectl capsule %s tenant oil

  # Uncordon a Namespace
  kubectl capsule %s ns oil-prod

  # Cordon a GlobalTenantResource
  kubectl capsule %s gtr default-policies

  # Uncordon a TenantResource in a namespace
  kubectl capsule %s tr custom-crds -n oil-prod`, verb, verb, verb, verb),
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

	return cmd
}
