// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/projectcapsule/capsule/cmd/cli/cmd/factory"
)

func TestNewRootCmd(t *testing.T) {
	t.Parallel()

	flags := genericclioptions.NewConfigFlags(true)
	f := factory.NewFactory(flags)
	streams, _, _, _ := genericclioptions.NewTestIOStreams()

	rootCmd := NewRootCmd(f, streams)
	assert.Equal(t, "capsule", rootCmd.Use)
	assert.True(t, rootCmd.HasSubCommands())

	subcommands := make(map[string]bool)
	for _, sub := range rootCmd.Commands() {
		subcommands[sub.Name()] = true
	}

	assert.True(t, subcommands["tenant"])
	assert.True(t, subcommands["namespace"])
	assert.True(t, subcommands["cordon"])
	assert.True(t, subcommands["uncordon"])
	assert.True(t, subcommands["reconcile"])
	assert.True(t, subcommands["expire"])
	assert.True(t, subcommands["resource-pool-claim"])
	assert.True(t, subcommands["resource-permit"])
	assert.True(t, subcommands["version"])
}
