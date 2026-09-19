// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
	"github.com/projectcapsule/capsule/pkg/api"
	"github.com/projectcapsule/capsule/pkg/api/rbac"
)

type CreateOptions struct {
	Factory   factory.Factory
	IOStreams genericclioptions.IOStreams

	Name                   string
	Owners                 []string
	NamespaceQuota         int32
	NodeSelectors          map[string]string
	AllowedRegistries      []string
	AllowedRegistriesRegex string

	Client ctrlclient.Client
}

func NewCmdCreate(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := &CreateOptions{
		Factory:   f,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "create NAME [flags]",
		Short: "Create a new Tenant",
		Args:  cobra.ExactArgs(1),
		Example: `  # Create a tenant with user owner
  kubectl capsule tenant create oil --owner User:alice

  # Create a tenant with quota and node selector
  kubectl capsule tenant create gas --owner User:bob --namespace-quota 5 --node-selector disk=ssd`,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Name = args[0]

			return o.Run(cmd.Context())
		},
	}

	cmd.Flags().StringArrayVar(&o.Owners, "owner", nil, "Owner of the tenant in format <Kind:Name> or <Name> (e.g. User:alice, Group:devs)")
	cmd.Flags().Int32Var(&o.NamespaceQuota, "namespace-quota", 0, "Hard quota of namespaces for this tenant")
	cmd.Flags().StringToStringVar(&o.NodeSelectors, "node-selector", nil, "Node selector key-value pairs (e.g. key=value)")
	cmd.Flags().StringArrayVar(&o.AllowedRegistries, "allowed-registries", nil, "Allowed container registries")
	cmd.Flags().StringVar(&o.AllowedRegistriesRegex, "allowed-registries-regex", "", "Allowed container registries regex")

	return cmd
}

func (o *CreateOptions) Run(ctx context.Context) error {
	var err error
	if o.Client == nil {
		o.Client, err = o.Factory.ToControllerRuntimeClient()
		if err != nil {
			return err
		}
	}

	tnt := &capsulev1beta2.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: o.Name,
		},
		Spec: capsulev1beta2.TenantSpec{},
	}

	for _, ownerStr := range o.Owners {
		parts := strings.SplitN(ownerStr, ":", 2)

		var (
			kind rbac.OwnerKind
			name string
		)

		if len(parts) == 2 {
			k := strings.ToLower(parts[0])
			switch k {
			case "user":
				kind = rbac.UserOwner
			case "group":
				kind = rbac.GroupOwner
			case "serviceaccount":
				kind = rbac.ServiceAccountOwner
			default:
				return fmt.Errorf("invalid owner kind %q; expected User, Group, or ServiceAccount", parts[0])
			}

			name = parts[1]
		} else {
			kind = rbac.UserOwner
			name = parts[0]
		}

		tnt.Spec.Owners = append(tnt.Spec.Owners, rbac.OwnerSpec{
			CoreOwnerSpec: rbac.CoreOwnerSpec{
				UserSpec: rbac.UserSpec{
					Kind: kind,
					Name: name,
				},
			},
		})
	}

	if o.NamespaceQuota > 0 {
		quota := o.NamespaceQuota
		tnt.Spec.NamespaceOptions = &capsulev1beta2.NamespaceOptions{
			Quota: &quota,
		}
	}

	if len(o.NodeSelectors) > 0 {
		tnt.Spec.NodeSelector = o.NodeSelectors
	}

	if len(o.AllowedRegistries) > 0 || o.AllowedRegistriesRegex != "" {
		tnt.Spec.ContainerRegistries = &api.AllowedListSpec{
			Exact: o.AllowedRegistries,
			Regex: o.AllowedRegistriesRegex,
		}
	}

	if err := o.Client.Create(ctx, tnt); err != nil {
		return fmt.Errorf("failed to create tenant %q: %w", o.Name, err)
	}

	_, _ = fmt.Fprintf(o.IOStreams.Out, "tenant.capsule.clastix.io/%s created\n", tnt.Name)

	return nil
}
