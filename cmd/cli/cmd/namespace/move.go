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
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/pkg/api/meta"
)

type MoveOptions struct {
	Factory   factory.Factory
	IOStreams genericiooptions.IOStreams

	Namespace    string
	TargetTenant string
	SourceTenant string
	Force        bool

	Client ctrlclient.Client
}

func NewCmdMove(f factory.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	o := &MoveOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:     "move NAME --to TARGET_TENANT [flags]",
		Aliases: []string{"migrate", "transfer"},
		Short:   "Move an existing Namespace from one Capsule Tenant to another",
		Args:    cobra.ExactArgs(1),
		Example: `  # Move namespace 'backend-dev' to tenant 'gas'
  kubectl capsule namespace move backend-dev --to gas

  # Move namespace verifying expected source tenant
  kubectl capsule namespace move backend-dev --from oil --to gas`,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Namespace = args[0]
			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&o.TargetTenant, "to", "", "Target tenant to migrate namespace to (required)")
	cmd.Flags().StringVar(&o.SourceTenant, "from", "", "Expected source tenant")
	cmd.Flags().BoolVarP(&o.Force, "force", "f", false, "Skip confirmation prompts")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func (o *MoveOptions) Run(ctx context.Context) error {
	var err error
	if o.Client == nil {
		o.Client, err = o.Factory.ToControllerRuntimeClient()
		if err != nil {
			return err
		}
	}

	if o.TargetTenant == "" {
		return fmt.Errorf("target tenant must be specified with --to")
	}

	targetTnt := &capsulev1beta2.Tenant{}
	if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.TargetTenant}, targetTnt); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("target tenant %q not found", o.TargetTenant)
		}
		return fmt.Errorf("failed to get target tenant %q: %w", o.TargetTenant, err)
	}

	if targetTnt.Spec.Cordoned {
		return fmt.Errorf("cannot move namespace to tenant %q because the target tenant is cordoned", o.TargetTenant)
	}

	if targetTnt.Spec.NamespaceOptions != nil && targetTnt.Spec.NamespaceOptions.Quota != nil {
		currentSize := len(targetTnt.Status.Spaces)
		if targetTnt.Status.Size > 0 && currentSize == 0 {
			currentSize = int(targetTnt.Status.Size)
		}
		if int32(currentSize) >= *targetTnt.Spec.NamespaceOptions.Quota {
			return fmt.Errorf("cannot move namespace to tenant %q: namespace quota (%d) exceeded",
				o.TargetTenant, *targetTnt.Spec.NamespaceOptions.Quota)
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
	if currentTenant == o.TargetTenant {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "namespace/%s is already in tenant %s\n", ns.Name, o.TargetTenant)
		return nil
	}

	if o.SourceTenant != "" && currentTenant != o.SourceTenant {
		return fmt.Errorf("namespace %q is assigned to tenant %q, but --from specified %q",
			o.Namespace, currentTenant, o.SourceTenant)
	}

	patch := ctrlclient.MergeFrom(ns.DeepCopy())

	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels[meta.TenantLabel] = targetTnt.Name

	ownerRef := metav1.OwnerReference{
		APIVersion:         capsulev1beta2.GroupVersion.String(),
		Kind:               "Tenant",
		Name:               targetTnt.Name,
		UID:                targetTnt.UID,
		BlockOwnerDeletion: ptr.To(true),
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

	_, _ = fmt.Fprintf(o.IOStreams.Out, "namespace/%s moved to tenant %s\n", ns.Name, o.TargetTenant)
	return nil
}
