// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package cordon

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
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

func TestCordonTenant(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	tnt := &capsulev1beta2.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "oil"},
		Spec:       capsulev1beta2.TenantSpec{Cordoned: false},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tnt).Build()

	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	var buf bytes.Buffer
	streams.Out = &buf

	opts := &CordonOptions{
		IOStreams:       streams,
		Client:          fakeClient,
		DesiredCordoned: true,
		ResourceType:    "tenant",
		ResourceName:    "oil",
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "tenant.capsule.clastix.io/oil cordoned")

	check := &capsulev1beta2.Tenant{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "oil"}, check))
	assert.True(t, check.Spec.Cordoned)
}

func TestCordonNamespace(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oil-prod",
			Labels: map[string]string{
				meta.TenantLabel: "oil",
			},
		},
	}
	tnt := &capsulev1beta2.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "oil"},
		Spec:       capsulev1beta2.TenantSpec{Cordoned: true},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, tnt).Build()

	t.Run("cordon namespace", func(t *testing.T) {
		streams, _, out, _ := genericclioptions.NewTestIOStreams()
		var buf bytes.Buffer
		streams.Out = &buf

		opts := &CordonOptions{
			IOStreams:       streams,
			Client:          fakeClient,
			DesiredCordoned: true,
			ResourceType:    "ns",
			ResourceName:    "oil-prod",
		}

		err := opts.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "namespace/oil-prod cordoned")

		check := &corev1.Namespace{}
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "oil-prod"}, check))
		assert.Equal(t, meta.ValueTrue, check.Labels[meta.CordonedLabel])
	})

	t.Run("uncordon namespace with warning for cordoned parent tenant", func(t *testing.T) {
		streams, _, out, errOut := genericclioptions.NewTestIOStreams()
		var buf bytes.Buffer
		var errBuf bytes.Buffer
		streams.Out = &buf
		streams.ErrOut = &errBuf

		opts := &CordonOptions{
			IOStreams:       streams,
			Client:          fakeClient,
			DesiredCordoned: false,
			ResourceType:    "namespace",
			ResourceName:    "oil-prod",
		}

		err := opts.Run(context.Background())
		require.NoError(t, err)
		assert.Contains(t, buf.String(), "namespace/oil-prod uncordoned")
		assert.Contains(t, errBuf.String(), `warning: parent tenant "oil" is cordoned`)

		check := &corev1.Namespace{}
		require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "oil-prod"}, check))
		assert.NotContains(t, check.Labels, meta.CordonedLabel)
	})
}

func TestCordonGlobalTenantResource(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	gtr := &capsulev1beta2.GlobalTenantResource{
		ObjectMeta: metav1.ObjectMeta{Name: "default-policies"},
		Spec: capsulev1beta2.GlobalTenantResourceSpec{
			TenantResourceCommonSpec: capsulev1beta2.TenantResourceCommonSpec{
				Cordoned: ptr.To(false),
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gtr).Build()

	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	var buf bytes.Buffer
	streams.Out = &buf

	opts := &CordonOptions{
		IOStreams:       streams,
		Client:          fakeClient,
		DesiredCordoned: true,
		ResourceType:    "gtr",
		ResourceName:    "default-policies",
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "globaltenantresource.capsule.clastix.io/default-policies cordoned")

	check := &capsulev1beta2.GlobalTenantResource{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "default-policies"}, check))
	assert.True(t, check.Spec.IsCordoned())
}

func TestCordonTenantResource(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	tr := &capsulev1beta2.TenantResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-policies",
			Namespace: "oil-dev",
		},
		Spec: capsulev1beta2.TenantResourceSpec{
			TenantResourceCommonSpec: capsulev1beta2.TenantResourceCommonSpec{
				Cordoned: ptr.To(false),
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tr).Build()

	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	var buf bytes.Buffer
	streams.Out = &buf

	opts := &CordonOptions{
		IOStreams:       streams,
		Client:          fakeClient,
		DesiredCordoned: true,
		ResourceType:    "tr",
		ResourceName:    "local-policies",
		Namespace:       "oil-dev",
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "tenantresource.capsule.clastix.io/local-policies cordoned")

	check := &capsulev1beta2.TenantResource{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{Name: "local-policies", Namespace: "oil-dev"}, check))
	assert.True(t, check.Spec.IsCordoned())
}
