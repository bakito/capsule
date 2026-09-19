// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepoolclaim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestResourcePoolClaimExpire(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	claim := &capsulev1beta2.ResourcePoolClaim{
		Name:      "my-claim",
		Namespace: "dev-tenant",
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(claim).
		Build()

	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	opts := &ExpireOptions{
		IOStreams: streams,
		Name:      "my-claim",
		Namespace: "dev-tenant",
		Client:    fakeClient,
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)

	assert.Contains(t, out.String(), "resourcepoolclaim.capsule.clastix.io/my-claim expired")

	updated := &capsulev1beta2.ResourcePoolClaim{}
	err = fakeClient.Get(context.Background(), ctrlclient.ObjectKey{Namespace: "dev-tenant", Name: "my-claim"}, updated)
	require.NoError(t, err)

	anno := updated.GetAnnotations()
	require.NotNil(t, anno)
	assert.Equal(t, meta.ReleaseAnnotationTrigger, anno[meta.ReleaseAnnotation])
}

func TestResourcePoolClaimExpireNotFound(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	opts := &ExpireOptions{
		IOStreams: streams,
		Name:      "non-existent",
		Namespace: "dev-tenant",
		Client:    fakeClient,
	}

	err := opts.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), `resourcepoolclaim "non-existent" in namespace "dev-tenant" not found`)
}

func TestResourcePoolClaimExpireValidate(t *testing.T) {
	t.Parallel()

	opts := &ExpireOptions{
		Name: "",
	}
	err := opts.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resourcepoolclaim name cannot be empty")
}
