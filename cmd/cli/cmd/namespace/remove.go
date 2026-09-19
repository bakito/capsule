// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package namespace

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/pkg/api/meta"
)

type RemoveOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams

	Namespace  string
	TenantName string

	Client ctrlclient.Client
}

func NewCmdRemove(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := &RemoveOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:     "remove NAME [flags]",
		Aliases: []string{"detach", "leave", "offboard"},
		Short:   "Remove a Namespace from Tenant management",
		Args:    cobra.ExactArgs(1),
		Example: `  # Remove namespace 'backend-dev' from its current tenant
  kubectl capsule namespace remove backend-dev

  # Remove namespace verifying expected tenant
  kubectl capsule namespace remove backend-dev --tenant oil`,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Namespace = args[0]

			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.TenantName, "tenant", "t", "", "Expected tenant the namespace belongs to")

	return cmd
}

func (o *RemoveOptions) Run(ctx context.Context) error {
	var err error
	if o.Client == nil && o.Factory != nil {
		o.Client, err = o.Factory.ToControllerRuntimeClient()
		if err != nil {
			return err
		}
	}

	ns := &corev1.Namespace{}
	if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.Namespace}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("namespace %q not found", o.Namespace)
		}

		return fmt.Errorf("failed to get namespace %q: %w", o.Namespace, err)
	}

	currentTenant := ns.Labels[meta.TenantLabel]
	if currentTenant == "" && len(ns.OwnerReferences) == 0 {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "namespace/%s is not assigned to any tenant\n", ns.Name)

		return nil
	}

	if o.TenantName != "" && currentTenant != o.TenantName {
		return fmt.Errorf("namespace %q is assigned to tenant %q, but --tenant specified %q",
			o.Namespace, currentTenant, o.TenantName)
	}

	patch := ctrlclient.MergeFrom(ns.DeepCopy())

	delete(ns.Labels, meta.TenantLabel)

	var newOwnerRefs []metav1.OwnerReference

	for _, ref := range ns.OwnerReferences {
		if ref.Kind != "Tenant" {
			newOwnerRefs = append(newOwnerRefs, ref)
		}
	}

	ns.OwnerReferences = newOwnerRefs

	if err := o.Client.Patch(ctx, ns, patch); err != nil {
		return fmt.Errorf("failed to patch namespace %q: %w", o.Namespace, err)
	}

	if currentTenant != "" {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "namespace/%s removed from tenant %s\n", ns.Name, currentTenant)
	} else {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "namespace/%s detached from tenant\n", ns.Name)
	}

	return nil
}
