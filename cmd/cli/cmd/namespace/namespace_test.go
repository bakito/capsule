// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package namespace

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/pkg/api/meta"
)

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = capsulev1beta2.AddToScheme(scheme)
	return scheme
}

func TestNamespaceAdd(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	quota := int32(2)
	tnt := &capsulev1beta2.Tenant{
		Name: "oil",
		UID:  types.UID("tnt-oil-uid"),
		Spec: capsulev1beta2.TenantSpec{
			NamespaceOptions: &capsulev1beta2.NamespaceOptions{
				Quota: &quota,
			},
		},
		Status: capsulev1beta2.TenantStatus{
			Spaces: []*capsulev1beta2.TenantStatusNamespaceItem{
				{Name: "oil-prod"},
			},
		},
	}
	ns := &corev1.Namespace{
		Name: "standalone-ns",
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tnt, ns).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	var buf bytes.Buffer
	streams.Out = &buf

	opts := &AddOptions{
		IOStreams:  streams,
		Client:     fakeClient,
		Namespace:  "standalone-ns",
		TenantName: "oil",
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "namespace/standalone-ns added to tenant oil")

	updated := &corev1.Namespace{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "standalone-ns"}, updated))
	assert.Equal(t, "oil", updated.Labels[meta.TenantLabel])
	require.Len(t, updated.OwnerReferences, 1)
	assert.Equal(t, "Tenant", updated.OwnerReferences[0].Kind)
	assert.Equal(t, "oil", updated.OwnerReferences[0].Name)
	assert.Equal(t, types.UID("tnt-oil-uid"), updated.OwnerReferences[0].UID)
}

func TestNamespaceAddErrors(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	quota := int32(1)
	cordonedTnt := &capsulev1beta2.Tenant{
		Name: "gas",
		Spec: capsulev1beta2.TenantSpec{Cordoned: true},
	}
	quotaFullTnt := &capsulev1beta2.Tenant{
		Name: "solar",
		Spec: capsulev1beta2.TenantSpec{
			NamespaceOptions: &capsulev1beta2.NamespaceOptions{Quota: &quota},
		},
		Status: capsulev1beta2.TenantStatus{
			Spaces: []*capsulev1beta2.TenantStatusNamespaceItem{{Name: "solar-ns1"}},
		},
	}
	ns := &corev1.Namespace{
		Name: "app-ns",
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cordonedTnt, quotaFullTnt, ns).
		Build()

	t.Run("target tenant cordoned", func(t *testing.T) {
		streams, _, _, _ := genericclioptions.NewTestIOStreams()
		opts := &AddOptions{
			IOStreams:  streams,
			Client:     fakeClient,
			Namespace:  "app-ns",
			TenantName: "gas",
		}
		err := opts.Run(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), `cannot add namespace to tenant "gas" because the tenant is cordoned`)
	})

	t.Run("quota exceeded", func(t *testing.T) {
		streams, _, _, _ := genericclioptions.NewTestIOStreams()
		opts := &AddOptions{
			IOStreams:  streams,
			Client:     fakeClient,
			Namespace:  "app-ns",
			TenantName: "solar",
		}
		err := opts.Run(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), `namespace quota (1) exceeded`)
	})
}

func TestNamespaceMove(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	tenantA := &capsulev1beta2.Tenant{
		Name: "tenant-a",
		UID:  types.UID("tnt-a-uid"),
	}
	tenantB := &capsulev1beta2.Tenant{
		Name: "tenant-b",
		UID:  types.UID("tnt-b-uid"),
	}
	ns := &corev1.Namespace{
		Name: "app-dev",
		Labels: map[string]string{
			meta.TenantLabel: "tenant-a",
		},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: capsulev1beta2.GroupVersion.String(),
				Kind:       "Tenant",
				Name:       "tenant-a",
				UID:        types.UID("tnt-a-uid"),
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tenantA, tenantB, ns).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	var buf bytes.Buffer
	streams.Out = &buf

	opts := &MoveOptions{
		IOStreams:    streams,
		Client:       fakeClient,
		Namespace:    "app-dev",
		SourceTenant: "tenant-a",
		TargetTenant: "tenant-b",
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "namespace/app-dev moved to tenant tenant-b")

	updated := &corev1.Namespace{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "app-dev"}, updated))
	assert.Equal(t, "tenant-b", updated.Labels[meta.TenantLabel])
	require.Len(t, updated.OwnerReferences, 1)
	assert.Equal(t, "tenant-b", updated.OwnerReferences[0].Name)
	assert.Equal(t, types.UID("tnt-b-uid"), updated.OwnerReferences[0].UID)
}

func TestNamespaceRemove(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	ns := &corev1.Namespace{
		Name: "app-prod",
		Labels: map[string]string{
			meta.TenantLabel: "tenant-a",
			"env":            "prod",
		},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: capsulev1beta2.GroupVersion.String(),
				Kind:       "Tenant",
				Name:       "tenant-a",
				UID:        types.UID("tnt-a-uid"),
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	var buf bytes.Buffer
	streams.Out = &buf

	opts := &RemoveOptions{
		IOStreams:  streams,
		Client:     fakeClient,
		Namespace:  "app-prod",
		TenantName: "tenant-a",
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "namespace/app-prod removed from tenant tenant-a")

	updated := &corev1.Namespace{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "app-prod"}, updated))
	assert.NotContains(t, updated.Labels, meta.TenantLabel)
	assert.Equal(t, "prod", updated.Labels["env"])
	assert.Empty(t, updated.OwnerReferences)
}
