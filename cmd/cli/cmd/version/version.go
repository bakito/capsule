// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"fmt"

	capsuleversion "github.com/projectcapsule/capsule/internal/version"
	"github.com/spf13/cobra"
)

// NewCmdVersion returns the version subcommand.
func NewCmdVersion(streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the Capsule CLI version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(streams.Out, "Capsule CLI version: %s (commit: %s, dirty: %s)\n",
				capsuleversion.GitTag, capsuleversion.GitCommit, capsuleversion.GitDirty)
		},
	}

	return cmd
}
