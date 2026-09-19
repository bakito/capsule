// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package resourcepermit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	apiruntime "github.com/projectcapsule/capsule/pkg/api/runtime"
)

func TestRenderedResourceRows(t *testing.T) {
	t.Parallel()

	rows := renderedResourceRows(apiruntime.RenderedResource{
		Policy: apiruntime.ResourceTemplatePolicy{
			Creation: apiruntime.ResourceCreationPolicyOwner,
			Deletion: apiruntime.ResourceDeletionPolicyRemove,
		},
		Targets: []runtime.RawExtension{
			{Raw: []byte(`{"kind":"RoleBinding","name":"alice"}`)},
			{Raw: []byte(`{"kind":"RoleBinding","name":"bob"}`)},
		},
	}, false)

	require.Len(t, rows, 2)
	assert.Contains(t, rows[0][0], "creation: Owner")
	assert.Contains(t, rows[0][1], "alice")
	assert.Contains(t, rows[1][1], "bob")
}

func TestPatchResourcePermitStatusPreservesControllerManagedFields(t *testing.T) {
	t.Parallel()

	permit := &capsulev1beta2.ResourcePermit{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "grant-admin",
			Namespace:  "default",
			Generation: 3,
		},
		Status: capsulev1beta2.ResourcePermitStatus{
			Phase:   capsulev1beta2.ResourcePermitPhaseRequested,
			Request: &capsulev1beta2.ResourcePermitStatusRequest{},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&capsulev1beta2.ResourcePermit{}).
		WithObjects(permit).
		Build()

	current := &capsulev1beta2.ResourcePermit{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      "grant-admin",
	}, current))

	err := patchResourcePermitStatus(context.Background(), fakeClient, current, func() error {
		current.Status.Phase = capsulev1beta2.ResourcePermitPhaseApproved
		current.Status.Review = &capsulev1beta2.ReviewInfo{
			Message: "LGTM",
		}

		return nil
	})
	require.NoError(t, err)

	updated := &capsulev1beta2.ResourcePermit{}
	require.NoError(t, fakeClient.Get(context.Background(), client.ObjectKey{
		Namespace: "default",
		Name:      "grant-admin",
	}, updated))

	assert.Equal(t, capsulev1beta2.ResourcePermitPhaseApproved, updated.Status.Phase)
	require.NotNil(t, updated.Status.Request)
	require.NotNil(t, updated.Status.Review)
	assert.Equal(t, "LGTM", updated.Status.Review.Message)
}
