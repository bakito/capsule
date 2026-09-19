// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package expire

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

func TestExpireResourcePoolClaim(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	claim := &capsulev1beta2.ResourcePoolClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-claim",
			Namespace: "dev-tenant",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(claim).
		Build()

	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	opts := &ExpireOptions{
		IOStreams:    streams,
		ResourceType: "rpc",
		ResourceName: "my-claim",
		Namespace:    "dev-tenant",
		Client:       fakeClient,
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

func TestExpireResourcePermit(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	permit := &capsulev1beta2.ResourcePermit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grant-access",
			Namespace: "default",
		},
		Status: capsulev1beta2.ResourcePermitStatus{
			Phase: capsulev1beta2.ResourcePermitPhaseActive,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&capsulev1beta2.ResourcePermit{}).
		WithObjects(permit).
		Build()

	streams, _, out, _ := genericclioptions.NewTestIOStreams()
	opts := &ExpireOptions{
		IOStreams:    streams,
		ResourceType: "resource-permit",
		ResourceName: "grant-access",
		Namespace:    "default",
		Client:       fakeClient,
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)

	assert.Contains(t, out.String(), "resourcepermit.capsule.clastix.io/grant-access expired")

	updated := &capsulev1beta2.ResourcePermit{}
	err = fakeClient.Get(context.Background(), ctrlclient.ObjectKey{Namespace: "default", Name: "grant-access"}, updated)
	require.NoError(t, err)
	assert.Equal(t, capsulev1beta2.ResourcePermitPhaseExpired, updated.Status.Phase)
}

func TestExpireTypeSlashName(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	claim := &capsulev1beta2.ResourcePoolClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "team-claim",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(claim).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()
	opts := &ExpireOptions{
		IOStreams: streams,
		Client:    fakeClient,
	}

	err := opts.Complete(nil, []string{"rpc/team-claim"})
	require.NoError(t, err)
	assert.Equal(t, "rpc", opts.ResourceType)
	assert.Equal(t, "team-claim", opts.ResourceName)

	err = opts.Validate()
	require.NoError(t, err)

	err = opts.Run(context.Background())
	require.NoError(t, err)
}

func TestExpireValidation(t *testing.T) {
	t.Parallel()

	opts := &ExpireOptions{
		ResourceType: "invalid",
		ResourceName: "something",
	}
	err := opts.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource type")

	optsEmpty := &ExpireOptions{
		ResourceType: "rpc",
		ResourceName: "",
	}
	err = optsEmpty.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource name cannot be empty")
}
