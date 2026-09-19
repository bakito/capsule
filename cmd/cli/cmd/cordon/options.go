// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package cordon

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/pkg/api/meta"
)

type CordonOptions struct {
	Factory         factory.Factory
	IOStreams       genericclioptions.IOStreams
	DesiredCordoned bool

	ResourceType string
	ResourceName string
	Namespace    string

	Client ctrlclient.Client
}

func (o *CordonOptions) Complete(cmd *cobra.Command, args []string) error {
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

	return fmt.Errorf("requires resource type and resource name (e.g. 'tenant oil' or 'tenant/oil')")
}

func (o *CordonOptions) Validate() error {
	if o.ResourceName == "" {
		return fmt.Errorf("resource name cannot be empty")
	}

	switch o.ResourceType {
	case "tenant", "tenants", "tnt",
		"namespace", "namespaces", "ns",
		"globaltenantresource", "globaltenantresources", "gtr",
		"tenantresource", "tenantresources", "tr":
		return nil
	default:
		return fmt.Errorf("unsupported resource type %q (supported: tenant, namespace, globaltenantresource, tenantresource)", o.ResourceType)
	}
}

func (o *CordonOptions) Run(ctx context.Context) error {
	action := "cordoned"
	if !o.DesiredCordoned {
		action = "uncordoned"
	}

	switch o.ResourceType {
	case "tenant", "tenants", "tnt":
		return o.runTenant(ctx, action)
	case "namespace", "namespaces", "ns":
		return o.runNamespace(ctx, action)
	case "globaltenantresource", "globaltenantresources", "gtr":
		return o.runGTR(ctx, action)
	case "tenantresource", "tenantresources", "tr":
		return o.runTR(ctx, action)
	default:
		return fmt.Errorf("unsupported resource type: %s", o.ResourceType)
	}
}

func (o *CordonOptions) runTenant(ctx context.Context, action string) error {
	tnt := &capsulev1beta2.Tenant{}
	if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.ResourceName}, tnt); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tenant %q not found", o.ResourceName)
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
		return fmt.Errorf("failed to %s tenant %q: %w", action[:len(action)-2], o.ResourceName, err)
	}

	_, _ = fmt.Fprintf(o.IOStreams.Out, "tenant.capsule.clastix.io/%s %s\n", tnt.Name, action)

	return nil
}

func (o *CordonOptions) runNamespace(ctx context.Context, action string) error {
	ns := &corev1.Namespace{}
	if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.ResourceName}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("namespace %q not found", o.ResourceName)
		}

		return err
	}

	isCordoned := ns.Labels[meta.CordonedLabel] == meta.ValueTrue

	if isCordoned == o.DesiredCordoned {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "namespace/%s already %s\n", ns.Name, action)

		return nil
	}

	if !o.DesiredCordoned {
		// Warn if parent tenant is cordoned
		if tenantName, ok := ns.Labels[meta.TenantLabel]; ok && tenantName != "" {
			parentTnt := &capsulev1beta2.Tenant{}
			if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: tenantName}, parentTnt); err == nil && parentTnt.Spec.Cordoned {
				_, _ = fmt.Fprintf(o.IOStreams.ErrOut,
					"warning: parent tenant %q is cordoned; namespace %q will continue to reject workload modifications until the tenant is uncordoned.\n",
					tenantName, ns.Name)
			}
		}
	}

	patch := ctrlclient.MergeFrom(ns.DeepCopy())

	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}

	if o.DesiredCordoned {
		ns.Labels[meta.CordonedLabel] = meta.ValueTrue
	} else {
		delete(ns.Labels, meta.CordonedLabel)
	}

	if err := o.Client.Patch(ctx, ns, patch); err != nil {
		return fmt.Errorf("failed to %s namespace %q: %w", action[:len(action)-2], o.ResourceName, err)
	}

	_, _ = fmt.Fprintf(o.IOStreams.Out, "namespace/%s %s\n", ns.Name, action)

	return nil
}

func (o *CordonOptions) runGTR(ctx context.Context, action string) error {
	gtr := &capsulev1beta2.GlobalTenantResource{}
	if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.ResourceName}, gtr); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("globaltenantresource %q not found", o.ResourceName)
		}

		return err
	}

	if gtr.Spec.IsCordoned() == o.DesiredCordoned {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "globaltenantresource.capsule.clastix.io/%s already %s\n", gtr.Name, action)

		return nil
	}

	patch := ctrlclient.MergeFrom(gtr.DeepCopy())
	gtr.Spec.Cordoned = new(o.DesiredCordoned)

	if err := o.Client.Patch(ctx, gtr, patch); err != nil {
		return fmt.Errorf("failed to %s globaltenantresource %q: %w", action[:len(action)-2], o.ResourceName, err)
	}

	_, _ = fmt.Fprintf(o.IOStreams.Out, "globaltenantresource.capsule.clastix.io/%s %s\n", gtr.Name, action)

	return nil
}

func (o *CordonOptions) runTR(ctx context.Context, action string) error {
	tr := &capsulev1beta2.TenantResource{}
	if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.ResourceName, Namespace: o.Namespace}, tr); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tenantresource %s/%s not found", o.Namespace, o.ResourceName)
		}

		return err
	}

	if tr.Spec.IsCordoned() == o.DesiredCordoned {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "tenantresource.capsule.clastix.io/%s already %s\n", tr.Name, action)

		return nil
	}

	patch := ctrlclient.MergeFrom(tr.DeepCopy())
	tr.Spec.Cordoned = new(o.DesiredCordoned)

	if err := o.Client.Patch(ctx, tr, patch); err != nil {
		return fmt.Errorf("failed to %s tenantresource %q: %w", action[:len(action)-2], o.ResourceName, err)
	}

	_, _ = fmt.Fprintf(o.IOStreams.Out, "tenantresource.capsule.clastix.io/%s %s\n", tr.Name, action)

	return nil
}
