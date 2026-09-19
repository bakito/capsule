// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestVersionCmd(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	streams := genericclioptions.IOStreams{
		Out:    &buf,
		ErrOut: &buf,
	}
	cmd := NewCmdVersion(streams)

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Capsule CLI version:")
}
