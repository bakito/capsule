// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

type DescribeOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams

	Name   string
	Client ctrlclient.Client
}

func NewCmdDescribe(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := &DescribeOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:     "describe NAME",
		Short:   "Show details of a specific Tenant",
		Args:    cobra.ExactArgs(1),
		Example: "  # Describe tenant 'oil'\n  kubectl capsule tenant describe oil",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]

			return o.Run(cmd.Context())
		},
	}

	return cmd
}

func printNamespaceInfo(out io.Writer, tnt *capsulev1beta2.Tenant) {
	quota := "<unlimited>"
	if tnt.Spec.NamespaceOptions != nil && tnt.Spec.NamespaceOptions.Quota != nil {
		quota = fmt.Sprintf("%d", *tnt.Spec.NamespaceOptions.Quota)
	}

	_, _ = fmt.Fprintf(out, "Namespace Quota:\t%s\n", quota)

	nsCount := len(tnt.Status.Spaces)
	if tnt.Status.Size > 0 && len(tnt.Status.Spaces) == 0 {
		nsCount = int(tnt.Status.Size)
	}

	_, _ = fmt.Fprintf(out, "Namespace Count:\t%d\n", nsCount)

	if len(tnt.Status.Spaces) > 0 {
		_, _ = fmt.Fprintln(out, "Namespaces:")

		for _, s := range tnt.Status.Spaces {
			_, _ = fmt.Fprintf(out, "  - %s\n", s.Name)
		}
	}
}

func printOwnerInfo(out io.Writer, tnt *capsulev1beta2.Tenant) {
	if len(tnt.Spec.Owners) > 0 {
		_, _ = fmt.Fprintln(out, "Owners:")

		for _, owner := range tnt.Spec.Owners {
			_, _ = fmt.Fprintf(out, "  - Kind: %s, Name: %s\n", owner.Kind, owner.Name)
		}
	} else {
		_, _ = fmt.Fprintln(out, "Owners:\t\t\t<none>")
	}
}

func printPolicyInfo(out io.Writer, tnt *capsulev1beta2.Tenant) {
	if len(tnt.Spec.NodeSelector) > 0 {
		selectors := make([]string, 0, len(tnt.Spec.NodeSelector))

		for k, v := range tnt.Spec.NodeSelector {
			selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
		}

		sort.Strings(selectors)

		_, _ = fmt.Fprintf(out, "Node Selector:\t\t%s\n", strings.Join(selectors, ", "))
	} else {
		_, _ = fmt.Fprintln(out, "Node Selector:\t\t<none>")
	}

	if tnt.Spec.IngressOptions.AllowedHostnames != nil && len(tnt.Spec.IngressOptions.AllowedHostnames.Exact) > 0 {
		_, _ = fmt.Fprintf(out, "Allowed Hostnames:\t%s\n", strings.Join(tnt.Spec.IngressOptions.AllowedHostnames.Exact, ", "))
	}

	if tnt.Spec.IngressOptions.AllowedClasses != nil && len(tnt.Spec.IngressOptions.AllowedClasses.Exact) > 0 {
		_, _ = fmt.Fprintf(out, "Allowed Ingress Classes:%s\n", strings.Join(tnt.Spec.IngressOptions.AllowedClasses.Exact, ", "))
	}

	if tnt.Spec.StorageClasses != nil && len(tnt.Spec.StorageClasses.Exact) > 0 {
		_, _ = fmt.Fprintf(out, "Allowed Storage Classes:%s\n", strings.Join(tnt.Spec.StorageClasses.Exact, ", "))
	}

	//nolint:staticcheck // support deprecated container registries field for backwards compatibility
	if tnt.Spec.ContainerRegistries != nil {
		if len(tnt.Spec.ContainerRegistries.Exact) > 0 {
			_, _ = fmt.Fprintf(out, "Allowed Registries:\t%s\n", strings.Join(tnt.Spec.ContainerRegistries.Exact, ", "))
		}

		if tnt.Spec.ContainerRegistries.Regex != "" {
			_, _ = fmt.Fprintf(out, "Allowed Registries Regex:%s\n", tnt.Spec.ContainerRegistries.Regex)
		}
	}
}

func printMetadataInfo(out io.Writer, tnt *capsulev1beta2.Tenant) {
	if len(tnt.Labels) > 0 {
		_, _ = fmt.Fprintln(out, "Labels:")

		keys := make([]string, 0, len(tnt.Labels))

		for k := range tnt.Labels {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, k := range keys {
			_, _ = fmt.Fprintf(out, "  %s=%s\n", k, tnt.Labels[k])
		}
	}

	if len(tnt.Annotations) > 0 {
		_, _ = fmt.Fprintln(out, "Annotations:")

		keys := make([]string, 0, len(tnt.Annotations))

		for k := range tnt.Annotations {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, k := range keys {
			_, _ = fmt.Fprintf(out, "  %s: %s\n", k, tnt.Annotations[k])
		}
	}
}

func (o *DescribeOptions) Run(ctx context.Context) error {
	var err error
	if o.Client == nil && o.Factory != nil {
		o.Client, err = o.Factory.ToControllerRuntimeClient()
		if err != nil {
			return err
		}
	}

	tnt := &capsulev1beta2.Tenant{}
	if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.Name}, tnt); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tenant %q not found", o.Name)
		}

		return err
	}

	out := o.IOStreams.Out
	_, _ = fmt.Fprintf(out, "Name:\t\t\t%s\n", tnt.Name)

	state := "Active"
	if tnt.Spec.Cordoned {
		state = "Cordoned"
	}

	_, _ = fmt.Fprintf(out, "State:\t\t\t%s\n", state)

	printNamespaceInfo(out, tnt)
	printOwnerInfo(out, tnt)
	printPolicyInfo(out, tnt)
	printMetadataInfo(out, tnt)

	return nil
}
