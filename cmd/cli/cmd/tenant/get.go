// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/utils"
)

type GetOptions struct {
	Factory    factory.Factory
	IOStreams  genericiooptions.IOStreams
	Output     utils.OutputOptions
	ShowLabels bool
	Selector   string

	Name   string
	Client ctrlclient.Client
}

func NewCmdGet(f factory.Factory, streams genericiooptions.IOStreams) *cobra.Command {
	o := &GetOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:     "get [NAME]",
		Aliases: []string{"list"},
		Short:   "Display one or many Tenants",
		Args:    cobra.MaximumNArgs(1),
		Example: `  # List all Tenants
  kubectl capsule tenant get

  # List all Tenants in JSON/YAML format
  kubectl capsule tenant get -o json
  kubectl capsule tenant get -o yaml

  # List all Tenants with wide output (shows owners)
  kubectl capsule tenant get -o wide

  # Get a single Tenant
  kubectl capsule tenant get oil`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.Name = args[0]
			}
			return o.Run(cmd.Context())
		},
	}

	o.Output.AddFlags(cmd)
	cmd.Flags().BoolVar(&o.ShowLabels, "show-labels", false, "When printing, show all labels as the last column")
	cmd.Flags().StringVarP(&o.Selector, "selector", "l", "", "Selector (label query) to filter on")

	return cmd
}

func (o *GetOptions) Run(ctx context.Context) error {
	var err error
	if o.Client == nil {
		o.Client, err = o.Factory.ToControllerRuntimeClient()
		if err != nil {
			return err
		}
	}

	if o.Name != "" {
		tnt := &capsulev1beta2.Tenant{}
		if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.Name}, tnt); err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("tenant %q not found", o.Name)
			}
			return err
		}

		handled, err := o.Output.PrintObject(o.IOStreams.Out, tnt)
		if handled {
			return err
		}

		return o.printTenantsTable([]capsulev1beta2.Tenant{*tnt})
	}

	listOptions := &ctrlclient.ListOptions{}
	if o.Selector != "" {
		selector, err := labels.Parse(o.Selector)
		if err != nil {
			return fmt.Errorf("invalid selector %q: %w", o.Selector, err)
		}
		listOptions.LabelSelector = selector
	}

	tenantList := &capsulev1beta2.TenantList{}
	if err := o.Client.List(ctx, tenantList, listOptions); err != nil {
		return err
	}

	handled, err := o.Output.PrintObject(o.IOStreams.Out, tenantList)
	if handled {
		return err
	}

	if len(tenantList.Items) == 0 {
		if o.Selector != "" {
			_, _ = fmt.Fprintln(o.IOStreams.ErrOut, "No resources found matching selector.")
		} else {
			_, _ = fmt.Fprintln(o.IOStreams.ErrOut, "No resources found.")
		}
		return nil
	}

	return o.printTenantsTable(tenantList.Items)
}

func (o *GetOptions) printTenantsTable(tenants []capsulev1beta2.Tenant) error {
	w := utils.NewTabWriter(o.IOStreams.Out)
	defer func() { _ = w.Flush() }()

	headers := []string{"NAME", "STATE", "NAMESPACES", "NAMESPACE QUOTA", "NODE SELECTOR"}
	if o.Output.IsWide() {
		headers = append(headers, "OWNERS")
	}
	headers = append(headers, "AGE")
	if o.ShowLabels {
		headers = append(headers, "LABELS")
	}

	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))

	for _, t := range tenants {
		state := "Active"
		if t.Spec.Cordoned {
			state = "Cordoned"
		}

		nsCount := fmt.Sprintf("%d", len(t.Status.Spaces))
		if t.Status.Size > 0 && len(t.Status.Spaces) == 0 {
			nsCount = fmt.Sprintf("%d", t.Status.Size)
		}

		quota := "<none>"
		if t.Spec.NamespaceOptions != nil && t.Spec.NamespaceOptions.Quota != nil {
			quota = fmt.Sprintf("%d", *t.Spec.NamespaceOptions.Quota)
		}

		nodeSelector := "<none>"
		if len(t.Spec.NodeSelector) > 0 {
			var selectors []string
			for k, v := range t.Spec.NodeSelector {
				selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
			}
			sort.Strings(selectors)
			nodeSelector = strings.Join(selectors, ",")
		}

		age := utils.FormatAge(t.CreationTimestamp)

		row := []string{t.Name, state, nsCount, quota, nodeSelector}

		if o.Output.IsWide() {
			var owners []string
			for _, owner := range t.Spec.Owners {
				owners = append(owners, fmt.Sprintf("%s:%s", owner.Kind, owner.Name))
			}
			ownersStr := "<none>"
			if len(owners) > 0 {
				ownersStr = strings.Join(owners, ",")
			}
			row = append(row, ownersStr)
		}

		row = append(row, age)

		if o.ShowLabels {
			var labelList []string
			for k, v := range t.Labels {
				labelList = append(labelList, fmt.Sprintf("%s=%s", k, v))
			}
			sort.Strings(labelList)
			labelsStr := "<none>"
			if len(labelList) > 0 {
				labelsStr = strings.Join(labelList, ",")
			}
			row = append(row, labelsStr)
		}

		_, _ = fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	return nil
}
