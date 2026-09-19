// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepermit

import (
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(capsulev1beta2.AddToScheme(scheme))
}

// NewCmdResourcePermit returns the resource-permit parent command.
func NewCmdResourcePermit(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resource-permit",
		Aliases: []string{"resourcepermit", "rp", "permit"},
		Short:   "Manage ResourcePermits",
		Long:    "Get, review, activate, expire, and retry ResourcePermits.",
	}

	cmd.AddCommand(
		NewCmdGet(f, streams),
		NewCmdReview(f, streams),
		NewCmdActivate(f, streams),
		NewCmdExpire(f, streams),
		NewCmdRetry(f, streams),
	)

	return cmd
}
