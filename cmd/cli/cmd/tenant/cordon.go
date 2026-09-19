// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

type CordonOptions struct {
	Factory         factory.Factory
	IOStreams       genericclioptions.IOStreams
	DesiredCordoned bool

	Name   string
	Client ctrlclient.Client
}

func NewCmdTenantCordon(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	return newCmdTenantCordonToggle(f, streams, true)
}

func NewCmdTenantUncordon(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	return newCmdTenantCordonToggle(f, streams, false)
}

func newCmdTenantCordonToggle(f factory.Factory, streams genericclioptions.IOStreams, cordoned bool) *cobra.Command {
	verb := "cordon"
	short := "Cordon a Tenant to prevent new namespace operations"

	if !cordoned {
		verb = "uncordon"
		short = "Uncordon a Tenant to resume namespace operations"
	}

	o := &CordonOptions{
		Factory:         f,
		IOStreams:       streams,
		DesiredCordoned: cordoned,
	}

	cmd := &cobra.Command{
		Use:   verb + " NAME",
		Short: short,
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf(`  # %s tenant 'oil'
  kubectl capsule tenant %s oil`, verb, verb),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]

			return o.Run(cmd.Context())
		},
	}

	return cmd
}

func (o *CordonOptions) Run(ctx context.Context) error {
	var err error
	if o.Client == nil && o.Factory != nil {
		o.Client, err = o.Factory.ToControllerRuntimeClient()
		if err != nil {
			return err
		}
	}

	action := "cordoned"
	if !o.DesiredCordoned {
		action = "uncordoned"
	}

	tnt := &capsulev1beta2.Tenant{}
	if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.Name}, tnt); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tenant %q not found", o.Name)
		}

		return err
	}

	if tnt.Spec.Cordoned == o.DesiredCordoned {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "tenant.capsule.clastix.io/%s already %s\n", tnt.Name, action)

		return nil
	}

	patch := ctrlclient.MergeFrom(tnt.DeepCopy())
	tnt.Spec.Cordoned = o.DesiredCordoned

	if err := o.Client.Patch(ctx, tnt, patch); err != nil {
		return fmt.Errorf("failed to %s tenant %q: %w", action[:len(action)-2], o.Name, err)
	}

	_, _ = fmt.Fprintf(o.IOStreams.Out, "tenant.capsule.clastix.io/%s %s\n", tnt.Name, action)

	return nil
}
