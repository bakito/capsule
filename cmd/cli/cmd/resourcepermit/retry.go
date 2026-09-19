// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepermit

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

// NewCmdRetry returns the retry subcommand.
func NewCmdRetry(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	return newActionCmd(
		f,
		streams,
		capsulev1beta2.ResourcePermitPhaseRetrying,
		"retry",
		"Retry a failed ResourcePermit",
		"  # Retry a failed ResourcePermit after fixing its execution identity or permissions\n  kubectl capsule resource-permit retry grant-admin --namespace default",
	)
}
