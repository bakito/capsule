// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package expire

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/resourcepermit"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/resourcepoolclaim"
)

type ExpireOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams

	ResourceType string
	ResourceName string
	Namespace    string

	Client ctrlclient.Client
}

func (o *ExpireOptions) Complete(cmd *cobra.Command, args []string) error {
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

	return fmt.Errorf("requires resource type and resource name (e.g. 'resourcepoolclaim my-claim' or 'rpc/my-claim')")
}

func (o *ExpireOptions) Validate() error {
	if o.ResourceName == "" {
		return fmt.Errorf("resource name cannot be empty")
	}

	switch o.ResourceType {
	case "resourcepoolclaim", "resourcepoolclaims", "resource-pool-claim", "resource-pool-claims", "rpc", "claim",
		"resourcepermit", "resourcepermits", "resource-permit", "resource-permits", "rp", "permit":
		return nil
	default:
		return fmt.Errorf("unsupported resource type %q (supported: resourcepoolclaim, resourcepermit)", o.ResourceType)
	}
}

func (o *ExpireOptions) Run(ctx context.Context) error {
	switch o.ResourceType {
	case "resourcepoolclaim", "resourcepoolclaims", "resource-pool-claim", "resource-pool-claims", "rpc", "claim":
		return resourcepoolclaim.ExpireResourcePoolClaim(ctx, o.Client, o.Namespace, o.ResourceName, o.IOStreams.Out)
	case "resourcepermit", "resourcepermits", "resource-permit", "resource-permits", "rp", "permit":
		opts := &resourcepermit.ActionOptions{
			IOStreams: o.IOStreams,
			Name:      o.ResourceName,
			Namespace: o.Namespace,
			Phase:     capsulev1beta2.ResourcePermitPhaseExpired,
			Client:    o.Client,
		}

		if err := opts.Run(ctx); err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("resourcepermit %q in namespace %q not found", o.ResourceName, o.Namespace)
			}

			return err
		}

		_, _ = fmt.Fprintf(o.IOStreams.Out, "resourcepermit.capsule.clastix.io/%s expired\n", o.ResourceName)

		return nil
	default:
		return fmt.Errorf("unsupported resource type: %s", o.ResourceType)
	}
}
