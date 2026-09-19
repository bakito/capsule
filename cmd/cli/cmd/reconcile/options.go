// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package reconcile

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/pkg/api/meta"
)

type ReconcileOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams

	ResourceType string
	ResourceName string
	Namespace    string

	Client ctrlclient.Client
}

func (o *ReconcileOptions) Complete(cmd *cobra.Command, args []string) error {
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

	if len(args) == 1 {
		parts := strings.SplitN(args[0], "/", 2)
		if len(parts) == 2 {
			o.ResourceType = strings.ToLower(parts[0])
			o.ResourceName = parts[1]

			return nil
		}
	} else if len(args) >= 2 {
		o.ResourceType = strings.ToLower(args[0])
		o.ResourceName = args[1]

		return nil
	}

	return fmt.Errorf("requires resource type and resource name (e.g. 'gtr default-policies' or 'gtr/default-policies')")
}

func (o *ReconcileOptions) Validate() error {
	if o.ResourceName == "" {
		return fmt.Errorf("resource name cannot be empty")
	}

	switch o.ResourceType {
	case "globaltenantresource", "globaltenantresources", "gtr",
		"tenantresource", "tenantresources", "tr":
		return nil
	default:
		return fmt.Errorf("unsupported resource type %q (supported: globaltenantresource, tenantresource)", o.ResourceType)
	}
}

func (o *ReconcileOptions) Run(ctx context.Context) error {
	switch o.ResourceType {
	case "globaltenantresource", "globaltenantresources", "gtr":
		return o.runGTR(ctx)
	case "tenantresource", "tenantresources", "tr":
		return o.runTR(ctx)
	default:
		return fmt.Errorf("unsupported resource type: %s", o.ResourceType)
	}
}

func (o *ReconcileOptions) runGTR(ctx context.Context) error {
	gvk := capsulev1beta2.GroupVersion.WithKind("GlobalTenantResource")
	key := types.NamespacedName{Name: o.ResourceName}

	if err := meta.TriggerRequestReconcileAnnotation(ctx, o.Client, gvk, key); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("globaltenantresource %q not found", o.ResourceName)
		}

		return fmt.Errorf("failed to request reconcile for globaltenantresource %q: %w", o.ResourceName, err)
	}

	_, _ = fmt.Fprintf(o.IOStreams.Out, "globaltenantresource.capsule.clastix.io/%s reconcile requested\n", o.ResourceName)

	return nil
}

func (o *ReconcileOptions) runTR(ctx context.Context) error {
	gvk := capsulev1beta2.GroupVersion.WithKind("TenantResource")
	key := types.NamespacedName{Namespace: o.Namespace, Name: o.ResourceName}

	if err := meta.TriggerRequestReconcileAnnotation(ctx, o.Client, gvk, key); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tenantresource %q in namespace %q not found", o.ResourceName, o.Namespace)
		}

		return fmt.Errorf("failed to request reconcile for tenantresource %q in namespace %q: %w", o.ResourceName, o.Namespace, err)
	}

	_, _ = fmt.Fprintf(o.IOStreams.Out, "tenantresource.capsule.clastix.io/%s reconcile requested\n", o.ResourceName)

	return nil
}
