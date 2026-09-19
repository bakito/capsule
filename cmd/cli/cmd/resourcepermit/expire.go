// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepermit

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

// NewCmdExpire returns the expire subcommand.
func NewCmdExpire(f factory.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	return newActionCmd(
		f,
		streams,
		capsulev1beta2.ResourcePermitPhaseExpired,
		"expire",
		"Expire a ResourcePermit",
		"  # Expire an existing ResourcePermit\n  kubectl capsule resource-permit expire grant-admin --namespace default",
	)
}
