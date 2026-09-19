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

	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	cmd := NewCmdVersion(streams)

	cmd.SetOut(out)
	cmd.SetErr(out)

	var buf bytes.Buffer
	streams.Out = &buf

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Capsule CLI version:")
}
