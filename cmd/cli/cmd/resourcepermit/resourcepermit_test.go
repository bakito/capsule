// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepermit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
)

func TestResourcePermitReview(t *testing.T) {
	t.Parallel()

	rp := &capsulev1beta2.ResourcePermit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dev-access",
			Namespace: "default",
		},
		Status: capsulev1beta2.ResourcePermitStatus{
			Phase:   capsulev1beta2.ResourcePermitPhaseRequested,
			Request: &capsulev1beta2.ResourcePermitStatusRequest{},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&capsulev1beta2.ResourcePermit{}).
		WithObjects(rp).
		Build()

	streams, _, _, _ := genericclioptions.NewTestIOStreams()

	opts := &ReviewOptions{
		IOStreams: streams,
		Client:    fakeClient,
		Name:      "dev-access",
		Namespace: "default",
		Approve:   true,
		Message:   "approved for maintenance",
	}

	err := opts.Run(context.Background())
	require.NoError(t, err)

	check := &capsulev1beta2.ResourcePermit{}
	require.NoError(t, fakeClient.Get(context.Background(), ctrlclient.ObjectKey{Name: "dev-access", Namespace: "default"}, check))
	assert.Equal(t, capsulev1beta2.ResourcePermitPhaseApproved, check.Status.Phase)
	require.NotNil(t, check.Status.Review)
	assert.Equal(t, "approved for maintenance", check.Status.Review.Message)
}

func TestResourcePermitActions(t *testing.T) {
	t.Parallel()

	rp := &capsulev1beta2.ResourcePermit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-permit",
			Namespace: "default",
		},
		Status: capsulev1beta2.ResourcePermitStatus{
			Phase: capsulev1beta2.ResourcePermitPhaseApproved,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&capsulev1beta2.ResourcePermit{}).
		WithObjects(rp).
		Build()

	t.Run("activate", func(t *testing.T) {
		streams, _, _, _ := genericclioptions.NewTestIOStreams()
		opts := &ActionOptions{
			IOStreams: streams,
			Client:    fakeClient,
			Name:      "test-permit",
			Namespace: "default",
			Phase:     capsulev1beta2.ResourcePermitPhaseActive,
		}
		require.NoError(t, opts.Run(context.Background()))

		check := &capsulev1beta2.ResourcePermit{}
		require.NoError(t, fakeClient.Get(context.Background(), ctrlclient.ObjectKey{Name: "test-permit", Namespace: "default"}, check))
		assert.Equal(t, capsulev1beta2.ResourcePermitPhaseActive, check.Status.Phase)
	})

	t.Run("expire", func(t *testing.T) {
		streams, _, _, _ := genericclioptions.NewTestIOStreams()
		opts := &ActionOptions{
			IOStreams: streams,
			Client:    fakeClient,
			Name:      "test-permit",
			Namespace: "default",
			Phase:     capsulev1beta2.ResourcePermitPhaseExpired,
		}
		require.NoError(t, opts.Run(context.Background()))

		check := &capsulev1beta2.ResourcePermit{}
		require.NoError(t, fakeClient.Get(context.Background(), ctrlclient.ObjectKey{Name: "test-permit", Namespace: "default"}, check))
		assert.Equal(t, capsulev1beta2.ResourcePermitPhaseExpired, check.Status.Phase)
	})

	t.Run("retry", func(t *testing.T) {
		streams, _, _, _ := genericclioptions.NewTestIOStreams()
		opts := &ActionOptions{
			IOStreams: streams,
			Client:    fakeClient,
			Name:      "test-permit",
			Namespace: "default",
			Phase:     capsulev1beta2.ResourcePermitPhaseRetrying,
		}
		require.NoError(t, opts.Run(context.Background()))

		check := &capsulev1beta2.ResourcePermit{}
		require.NoError(t, fakeClient.Get(context.Background(), ctrlclient.ObjectKey{Name: "test-permit", Namespace: "default"}, check))
		assert.Equal(t, capsulev1beta2.ResourcePermitPhaseRetrying, check.Status.Phase)
	})
}
