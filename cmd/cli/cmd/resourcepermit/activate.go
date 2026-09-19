// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepermit

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/retry"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

type ActionOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams

	Name      string
	Namespace string
	Phase     capsulev1beta2.ResourcePermitPhase
	Client    ctrlclient.Client
}

func (o *ActionOptions) Run(ctx context.Context) error {
	var err error
	if o.Client == nil && o.Factory != nil {
		o.Client, err = o.Factory.ToControllerRuntimeClient()
		if err != nil {
			return err
		}
	}

	if o.Namespace == "" && o.Factory != nil {
		ns, _, _ := o.Factory.Namespace()
		o.Namespace = ns
	}

	if o.Namespace == "" {
		o.Namespace = "default"
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		br := &capsulev1beta2.ResourcePermit{}
		if err := o.Client.Get(
			ctx,
			ctrlclient.ObjectKey{Name: o.Name, Namespace: o.Namespace},
			br,
		); err != nil {
			return err
		}

		return patchResourcePermitStatus(ctx, o.Client, br, func() error {
			br.Status.Phase = o.Phase

			return nil
		})
	})
}

func newActionCmd(
	f factory.Factory,
	streams genericclioptions.IOStreams,
	phase capsulev1beta2.ResourcePermitPhase,
	use string,
	short string,
	example string,
) *cobra.Command {
	o := &ActionOptions{
		Factory:   f,
		IOStreams: streams,
		Phase:     phase,
	}

	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s NAME [flags]", use),
		Short:   short,
		Args:    cobra.ExactArgs(1),
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]

			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "Namespace of the ResourcePermit")

	return cmd
}

// NewCmdActivate returns the activate subcommand.
func NewCmdActivate(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	return newActionCmd(
		f,
		streams,
		capsulev1beta2.ResourcePermitPhaseActive,
		"activate",
		"Activate a ResourcePermit",
		"  # Activate an existing ResourcePermit\n  kubectl capsule resource-permit activate grant-admin --namespace default",
	)
}
