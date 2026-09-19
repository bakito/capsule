// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

func TestImpersonationOptionsApplyTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		options    ImpersonationOptions
		config     rest.Config
		wantUser   string
		wantUID    string
		wantGroups []string
		wantError  string
	}{
		{
			name:       "preserves kubeconfig impersonation",
			config:     rest.Config{Impersonate: rest.ImpersonationConfig{UserName: "configured", Groups: []string{"configured-group"}}},
			wantUser:   "configured",
			wantGroups: []string{"configured-group"},
		},
		{
			name:       "command flags override kubeconfig impersonation",
			options:    ImpersonationOptions{User: "alice", UID: "1001", Groups: []string{"developers", "on-call"}},
			config:     rest.Config{Impersonate: rest.ImpersonationConfig{UserName: "configured", Groups: []string{"configured-group"}}},
			wantUser:   "alice",
			wantUID:    "1001",
			wantGroups: []string{"developers", "on-call"},
		},
		{
			name:     "user flag clears kubeconfig impersonation groups",
			options:  ImpersonationOptions{User: "alice"},
			config:   rest.Config{Impersonate: rest.ImpersonationConfig{UserName: "configured", Groups: []string{"configured-group"}}},
			wantUser: "alice",
		},
		{
			name:       "group flags use kubeconfig impersonated user",
			options:    ImpersonationOptions{Groups: []string{"developers"}},
			config:     rest.Config{Impersonate: rest.ImpersonationConfig{UserName: "configured", Groups: []string{"configured-group"}}},
			wantUser:   "configured",
			wantGroups: []string{"developers"},
		},
		{
			name:       "groups require an impersonated user",
			options:    ImpersonationOptions{Groups: []string{"developers"}},
			wantGroups: []string{"developers"},
			wantError:  "--as-group requires --as or an impersonated user in the kubeconfig",
		},
		{
			name:      "uid requires an impersonated user",
			options:   ImpersonationOptions{UID: "1001"},
			wantUID:   "1001",
			wantError: "--as-group requires --as or an impersonated user in the kubeconfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.config
			err := tt.options.ApplyTo(&cfg)

			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantUser, cfg.Impersonate.UserName)
			assert.Equal(t, tt.wantUID, cfg.Impersonate.UID)
			assert.Equal(t, tt.wantGroups, cfg.Impersonate.Groups)
		})
	}
}

func TestFactoryImpersonationClients(t *testing.T) {
	t.Parallel()

	user := "alice"
	uid := "10001"
	groups := []string{"developers", "on-call"}

	flags := genericclioptions.NewConfigFlags(true)
	flags.Impersonate = &user
	flags.ImpersonateUID = &uid
	flags.ImpersonateGroup = &groups
	f := NewFactory(flags)

	// Verify ToRESTConfig() contains impersonation
	cfg, err := f.ToRESTConfig()
	require.NoError(t, err)
	assert.Equal(t, "alice", cfg.Impersonate.UserName)
	assert.Equal(t, "10001", cfg.Impersonate.UID)
	assert.Equal(t, []string{"developers", "on-call"}, cfg.Impersonate.Groups)

	// Verify ToControllerRuntimeClient() succeeds with the impersonated config
	ctrlClient, err := f.ToControllerRuntimeClient()
	require.NoError(t, err)
	assert.NotNil(t, ctrlClient)

	// Verify ToKubeClient() succeeds with the impersonated config
	kubeClient, err := f.ToKubeClient()
	require.NoError(t, err)
	assert.NotNil(t, kubeClient)

	// Verify ToDynamicClient() succeeds with the impersonated config
	dynClient, err := f.ToDynamicClient()
	require.NoError(t, err)
	assert.NotNil(t, dynClient)

	// Verify ToDiscoveryClient() succeeds
	discClient, err := f.ToDiscoveryClient()
	require.NoError(t, err)
	assert.NotNil(t, discClient)

	// Verify ConfigFlags() getter
	assert.Equal(t, flags, f.ConfigFlags())
}
