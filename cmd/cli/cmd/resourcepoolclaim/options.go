// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepoolclaim

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/retry"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/pkg/api/meta"
)

type ExpireOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams

	Name      string
	Namespace string

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

	if len(args) > 0 {
		o.Name = args[0]
	}

	return nil
}

func (o *ExpireOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("resourcepoolclaim name cannot be empty")
	}

	return nil
}

func (o *ExpireOptions) Run(ctx context.Context) error {
	return ExpireResourcePoolClaim(ctx, o.Client, o.Namespace, o.Name, o.IOStreams.Out)
}

// ExpireResourcePoolClaim patches the release annotation on the ResourcePoolClaim.
func ExpireResourcePoolClaim(ctx context.Context, c ctrlclient.Client, namespace, name string, out io.Writer) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		rpc := &capsulev1beta2.ResourcePoolClaim{}
		if err := c.Get(ctx, ctrlclient.ObjectKey{Namespace: namespace, Name: name}, rpc); err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("resourcepoolclaim %q in namespace %q not found", name, namespace)
			}

			return err
		}

		base := rpc.DeepCopy()

		annotations := rpc.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}

		annotations[meta.ReleaseAnnotation] = meta.ReleaseAnnotationTrigger

		rpc.SetAnnotations(annotations)

		return c.Patch(ctx, rpc, ctrlclient.MergeFrom(base))
	})
	if err != nil {
		return err
	}

	if out != nil {
		_, _ = fmt.Fprintf(out, "resourcepoolclaim.capsule.clastix.io/%s expired\n", name)
	}

	return nil
}
