// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package reconcile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/pkg/api/meta"
)

func newTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(capsulev1beta2.AddToScheme(scheme))
	return scheme
}

func TestReconcileGlobalTenantResource(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	gtr := &capsulev1beta2.GlobalTenantResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-policies",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gtr).
		Build()

	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	opts := &ReconcileOptions{
		IOStreams:    streams,
		ResourceType: "gtr",
		ResourceName: "default-policies",
		Client:       fakeClient,
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)

	assert.Contains(t, out.String(), "globaltenantresource.capsule.clastix.io/default-policies reconcile requested")

	updated := &capsulev1beta2.GlobalTenantResource{}
	err = fakeClient.Get(context.Background(), ctrlclient.ObjectKey{Name: "default-policies"}, updated)
	require.NoError(t, err)

	anno := updated.GetAnnotations()
	require.NotNil(t, anno)
	assert.NotEmpty(t, anno[meta.ReconcileAnnotation])
}

func TestReconcileTenantResource(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	tr := &capsulev1beta2.TenantResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-rules",
			Namespace: "oil-dev",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(tr).
		Build()

	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	opts := &ReconcileOptions{
		IOStreams:    streams,
		ResourceType: "tr",
		ResourceName: "local-rules",
		Namespace:    "oil-dev",
		Client:       fakeClient,
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)

	assert.Contains(t, out.String(), "tenantresource.capsule.clastix.io/local-rules reconcile requested")

	updated := &capsulev1beta2.TenantResource{}
	err = fakeClient.Get(context.Background(), ctrlclient.ObjectKey{Namespace: "oil-dev", Name: "local-rules"}, updated)
	require.NoError(t, err)

	anno := updated.GetAnnotations()
	require.NotNil(t, anno)
	assert.NotEmpty(t, anno[meta.ReconcileAnnotation])
}

func TestReconcileTypeSlashName(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	gtr := &capsulev1beta2.GlobalTenantResource{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sec-policies",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gtr).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	opts := &ReconcileOptions{
		IOStreams: streams,
		Client:    fakeClient,
	}

	err := opts.Complete(nil, []string{"gtr/sec-policies"})
	require.NoError(t, err)
	assert.Equal(t, "gtr", opts.ResourceType)
	assert.Equal(t, "sec-policies", opts.ResourceName)

	err = opts.Validate()
	require.NoError(t, err)

	err = opts.Run(context.Background())
	require.NoError(t, err)
}

func TestReconcileNotFound(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	opts := &ReconcileOptions{
		IOStreams:    streams,
		ResourceType: "gtr",
		ResourceName: "non-existent",
		Client:       fakeClient,
	}

	err := opts.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `globaltenantresource "non-existent" not found`)
}

func TestReconcileValidation(t *testing.T) {
	t.Parallel()

	opts := &ReconcileOptions{
		ResourceType: "invalid",
		ResourceName: "something",
	}
	err := opts.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource type")

	optsEmpty := &ReconcileOptions{
		ResourceType: "gtr",
		ResourceName: "",
	}
	err = optsEmpty.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource name cannot be empty")
}
