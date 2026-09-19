// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepermit

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
	"github.com/projectcapsule/capsule/cmd/cli/cmd/utils"
)

type GetOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams
	Output    utils.OutputOptions

	Name      string
	Namespace string
	Client    ctrlclient.Client
}

func NewCmdGet(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := &GetOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:     "get [NAME]",
		Aliases: []string{"list"},
		Short:   "Display one or many ResourcePermits",
		Args:    cobra.MaximumNArgs(1),
		Example: `  # List ResourcePermits in current/default namespace
  kubectl capsule resource-permit get

  # List ResourcePermits in a specific namespace
  kubectl capsule resource-permit get -n default

  # Get a single ResourcePermit
  kubectl capsule resource-permit get my-permit -n default`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.Name = args[0]
			}
			return o.Run(cmd.Context())
		},
	}

	o.Output.AddFlags(cmd)
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "Namespace of the ResourcePermit")

	return cmd
}

func (o *GetOptions) Run(ctx context.Context) error {
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

	if o.Name != "" {
		rp := &capsulev1beta2.ResourcePermit{}
		if err := o.Client.Get(ctx, ctrlclient.ObjectKey{Name: o.Name, Namespace: o.Namespace}, rp); err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("resourcepermit %s/%s not found", o.Namespace, o.Name)
			}
			return err
		}

		handled, err := o.Output.PrintObject(o.IOStreams.Out, rp)
		if handled {
			return err
		}

		return o.printResourcePermitsTable([]capsulev1beta2.ResourcePermit{*rp})
	}

	list := &capsulev1beta2.ResourcePermitList{}
	listOpts := []ctrlclient.ListOption{}
	if o.Namespace != "" {
		listOpts = append(listOpts, ctrlclient.InNamespace(o.Namespace))
	}

	if err := o.Client.List(ctx, list, listOpts...); err != nil {
		return err
	}

	handled, err := o.Output.PrintObject(o.IOStreams.Out, list)
	if handled {
		return err
	}

	if len(list.Items) == 0 {
		_, _ = fmt.Fprintln(o.IOStreams.ErrOut, "No resources found.")
		return nil
	}

	return o.printResourcePermitsTable(list.Items)
}

func (o *GetOptions) printResourcePermitsTable(items []capsulev1beta2.ResourcePermit) error {
	w := utils.NewTabWriter(o.IOStreams.Out)
	defer func() { _ = w.Flush() }()

	headers := []string{"NAME", "NAMESPACE", "PHASE", "REQUESTOR", "VALID FROM", "VALID UNTIL", "AGE"}
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))

	for _, item := range items {
		phase := string(item.Status.Phase)
		if phase == "" {
			phase = "<pending>"
		}

		requestor := "<none>"
		if item.Status.Request != nil && item.Status.Request.Impersonation != nil {
			requestor = item.Status.Request.Impersonation.Name
		}

		validFrom := "<none>"
		if item.Status.ValidFrom != nil {
			validFrom = item.Status.ValidFrom.Time.Format("2006-01-02 15:04:05")
		}

		validUntil := "<none>"
		if item.Status.ValidUntil != nil {
			validUntil = item.Status.ValidUntil.Time.Format("2006-01-02 15:04:05")
		}

		age := utils.FormatAge(item.CreationTimestamp)

		row := []string{item.Name, item.Namespace, phase, requestor, validFrom, validUntil, age}
		_, _ = fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	return nil
}
