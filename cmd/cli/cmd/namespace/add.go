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

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/pkg/api/meta"
)

type AddOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams

	Namespace  string
	TenantName string
	Client     ctrlclient.Client
}

func NewCmdAdd(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := &AddOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:     "add NAME --tenant TENANT",
		Aliases: []string{"join", "attach"},
		Short:   "Assign an existing Namespace to a Capsule Tenant",
		Args:    cobra.ExactArgs(1),
		Example: `  # Add namespace 'backend-dev' to tenant 'oil'
  kubectl capsule namespace add backend-dev --tenant oil`,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Namespace = args[0]

			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVarP(&o.TenantName, "tenant", "t", "", "Target tenant to assign the namespace to (required)")
	_ = cmd.MarkFlagRequired("tenant")

	return cmd
}

func (o *AddOptions) Run(ctx context.Context) error {
	var err error
	if o.Client == nil && o.Factory != nil {
		o.Client, err = o.Factory.ToControllerRuntimeClient()
		if err != nil {
			return err
		}
	}

	if o.TenantName == "" {
		return fmt.Errorf("target tenant name must be specified with --tenant")
	}

	tnt := &capsulev1beta2.Tenant{}
	if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.TenantName}, tnt); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tenant %q not found", o.TenantName)
		}

		return fmt.Errorf("failed to get tenant %q: %w", o.TenantName, err)
	}

	if tnt.Spec.Cordoned {
		return fmt.Errorf("cannot add namespace to tenant %q because the tenant is cordoned", o.TenantName)
	}

	if tnt.Spec.NamespaceOptions != nil && tnt.Spec.NamespaceOptions.Quota != nil {
		currentSize := len(tnt.Status.Spaces)
		if tnt.Status.Size > 0 && currentSize == 0 {
			currentSize = int(tnt.Status.Size)
		}

		if int64(currentSize) >= int64(*tnt.Spec.NamespaceOptions.Quota) {
			return fmt.Errorf("cannot add namespace to tenant %q: namespace quota (%d) exceeded",
				o.TenantName, *tnt.Spec.NamespaceOptions.Quota)
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
	if currentTenant == o.TenantName {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "namespace/%s is already added to tenant %s\n", ns.Name, o.TenantName)

		return nil
	}

	if currentTenant != "" {
		return fmt.Errorf("namespace %q is already part of tenant %q; use 'namespace move' to migrate between tenants",
			o.Namespace, currentTenant)
	}

	patch := ctrlclient.MergeFrom(ns.DeepCopy())

	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}

	ns.Labels[meta.TenantLabel] = tnt.Name

	ownerRef := metav1.OwnerReference{
		APIVersion:         capsulev1beta2.GroupVersion.String(),
		Kind:               "Tenant",
		Name:               tnt.Name,
		UID:                tnt.UID,
		BlockOwnerDeletion: new(true),
	}

	var newOwnerRefs []metav1.OwnerReference

	for _, ref := range ns.OwnerReferences {
		if ref.Kind != "Tenant" {
			newOwnerRefs = append(newOwnerRefs, ref)
		}
	}

	newOwnerRefs = append(newOwnerRefs, ownerRef)
	ns.OwnerReferences = newOwnerRefs

	if err := o.Client.Patch(ctx, ns, patch); err != nil {
		return fmt.Errorf("failed to patch namespace %q: %w", o.Namespace, err)
	}

	_, _ = fmt.Fprintf(o.IOStreams.Out, "namespace/%s added to tenant %s\n", ns.Name, o.TenantName)

	return nil
}
