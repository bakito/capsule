// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/pkg/api/rbac"
)

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = capsulev1beta2.AddToScheme(scheme)
	return scheme
}

func TestTenantDescribe(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	quota := int32(3)
	tnt := &capsulev1beta2.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: "solar",
			Labels: map[string]string{
				"region": "eu",
			},
		},
		Spec: capsulev1beta2.TenantSpec{
			Owners: rbac.OwnerListSpec{
				{
					CoreOwnerSpec: rbac.CoreOwnerSpec{
						UserSpec: rbac.UserSpec{
							Kind: rbac.UserOwner,
							Name: "charlie",
						},
					},
				},
			},
			NamespaceOptions: &capsulev1beta2.NamespaceOptions{
				Quota: &quota,
			},
			NodeSelector: map[string]string{
				"zone": "a",
			},
		},
		Status: capsulev1beta2.TenantStatus{
			Spaces: []*capsulev1beta2.TenantStatusNamespaceItem{
				{Name: "solar-ns1"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tnt).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	var buf bytes.Buffer
	streams.Out = &buf

	opts := &DescribeOptions{
		IOStreams: streams,
		Client:    fakeClient,
		Name:      "solar",
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)

	outStr := buf.String()
	assert.Contains(t, outStr, "Name:\t\t\tsolar")
	assert.Contains(t, outStr, "State:\t\t\tActive")
	assert.Contains(t, outStr, "Namespace Quota:\t3")
	assert.Contains(t, outStr, "solar-ns1")
	assert.Contains(t, outStr, "Kind: User, Name: charlie")
	assert.Contains(t, outStr, "zone=a")
}

func TestTenantCreate(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	var buf bytes.Buffer
	streams.Out = &buf

	opts := &CreateOptions{
		IOStreams:         streams,
		Client:            fakeClient,
		Name:              "wind",
		Owners:            []string{"User:dave", "Group:devops"},
		NamespaceQuota:    10,
		NodeSelectors:     map[string]string{"type": "spot"},
		AllowedRegistries: []string{"docker.io"},
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "tenant.capsule.clastix.io/wind created")

	created := &capsulev1beta2.Tenant{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "wind"}, created)
	require.NoError(t, err)
	assert.Equal(t, "wind", created.Name)
	require.Len(t, created.Spec.Owners, 2)
	assert.Equal(t, rbac.UserOwner, created.Spec.Owners[0].Kind)
	assert.Equal(t, "dave", created.Spec.Owners[0].Name)
	assert.Equal(t, rbac.GroupOwner, created.Spec.Owners[1].Kind)
	assert.Equal(t, "devops", created.Spec.Owners[1].Name)
	require.NotNil(t, created.Spec.NamespaceOptions)
	assert.Equal(t, int32(10), *created.Spec.NamespaceOptions.Quota)
	assert.Equal(t, map[string]string{"type": "spot"}, created.Spec.NodeSelector)
	require.NotNil(t, created.Spec.ContainerRegistries)
	assert.Equal(t, []string{"docker.io"}, created.Spec.ContainerRegistries.Exact)
}

func TestTenantCordonUncordon(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	tnt := &capsulev1beta2.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hydro",
		},
		Spec: capsulev1beta2.TenantSpec{
			Cordoned: false,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tnt).
		Build()

	t.Run("cordon tenant", func(t *testing.T) {
		streams, _, _, _ := genericclioptions.NewTestIOStreams()
		var buf bytes.Buffer
		streams.Out = &buf

		opts := &CordonOptions{
			IOStreams:       streams,
			Client:          fakeClient,
			Name:            "hydro",
			DesiredCordoned: true,
		}

		err := opts.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "tenant.capsule.clastix.io/hydro cordoned")

		check := &capsulev1beta2.Tenant{}
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "hydro"}, check))
		assert.True(t, check.Spec.Cordoned)
	})

	t.Run("cordon already cordoned tenant", func(t *testing.T) {
		streams, _, _, _ := genericclioptions.NewTestIOStreams()
		var buf bytes.Buffer
		streams.Out = &buf

		opts := &CordonOptions{
			IOStreams:       streams,
			Client:          fakeClient,
			Name:            "hydro",
			DesiredCordoned: true,
		}

		err := opts.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "tenant.capsule.clastix.io/hydro already cordoned")
	})

	t.Run("uncordon tenant", func(t *testing.T) {
		streams, _, _, _ := genericclioptions.NewTestIOStreams()
		var buf bytes.Buffer
		streams.Out = &buf

		opts := &CordonOptions{
			IOStreams:       streams,
			Client:          fakeClient,
			Name:            "hydro",
			DesiredCordoned: false,
		}

		err := opts.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "tenant.capsule.clastix.io/hydro uncordoned")

		check := &capsulev1beta2.Tenant{}
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "hydro"}, check))
		assert.False(t, check.Spec.Cordoned)
	})
}
