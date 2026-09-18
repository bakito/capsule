// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/pkg/api/rbac"
	"github.com/projectcapsule/capsule/pkg/runtime/configuration"
	"github.com/projectcapsule/capsule/pkg/runtime/events"
)

// BenchmarkNamespaceHandlerSubresourceUpdate measures the namespace validating
// chain for metadata writes carried by the plain resource and by the
// namespaces/status and namespaces/finalize subresources, which now run the
// full ownership and metadata handler chain instead of short-circuiting.
func BenchmarkNamespaceHandlerSubresourceUpdate(b *testing.B) {
	const ownerName = "alice"

	owner := rbac.CoreOwnerSpec{UserSpec: rbac.UserSpec{Name: ownerName, Kind: rbac.UserOwner}}

	for _, tenants := range []int{1, 50} {
		objects := make([]client.Object, 0, tenants+1)
		objects = append(objects, &capsulev1beta2.CapsuleConfiguration{
			ObjectMeta: metav1.ObjectMeta{Name: "capsule"},
			Status:     capsulev1beta2.CapsuleConfigurationStatus{Users: rbac.UserListSpec{owner.UserSpec}},
		})

		var solar *capsulev1beta2.Tenant

		for i := range tenants {
			tnt := &capsulev1beta2.Tenant{ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("tenant-%d", i),
				UID:  types.UID(fmt.Sprintf("uid-%d", i)),
			}}
			tnt.Status.Owners = rbac.OwnerStatusListSpec{owner}
			objects = append(objects, tnt)

			if i == 0 {
				solar = tnt
			}
		}

		scheme := namespaceValidationScheme(&testing.T{})
		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
		ctx := context.Background()
		cfg := configuration.NewCapsuleConfiguration(ctx, cl, cl, nil, "capsule")
		recorder := events.NewEventRecorder(nil, logr.Discard(), nil, nil)
		handler := NamespaceHandler(cfg, &recordingNamespaceHandler{}).OnUpdate(cl, cl, admission.NewDecoder(scheme), recorder)

		managed := namespaceWithTenantReference("solar-prod", solar.Name, string(solar.UID))
		labelled := managed.DeepCopy()
		labelled.Labels["probe"] = "true"
		unmanaged := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}
		unmanagedLabelled := unmanaged.DeepCopy()
		unmanagedLabelled.Labels = map[string]string{"probe": "true"}

		cases := []struct {
			name        string
			oldNs       *corev1.Namespace
			newNs       *corev1.Namespace
			subresource string
			wantDenied  bool
		}{
			{name: "plain/owner/allow", oldNs: managed, newNs: labelled},
			{name: "status/owner/allow", oldNs: managed, newNs: labelled, subresource: "status"},
			{name: "finalize/owner/allow", oldNs: managed, newNs: labelled, subresource: "finalize"},
			{name: "status/unmanaged/deny", oldNs: unmanaged, newNs: unmanagedLabelled, subresource: "status", wantDenied: true},
			{name: "finalize/unmanaged/deny", oldNs: unmanaged, newNs: unmanagedLabelled, subresource: "finalize", wantDenied: true},
		}

		for _, tc := range cases {
			b.Run(fmt.Sprintf("tenants=%d/%s", tenants, tc.name), func(b *testing.B) {
				req := namespaceUpdateRequest(&testing.T{}, tc.oldNs, tc.newNs, tc.subresource)
				req.UserInfo = authenticationv1.UserInfo{Username: ownerName}

				b.ReportAllocs()

				for b.Loop() {
					response := handler(ctx, req)

					denied := response != nil && !response.Allowed
					if denied != tc.wantDenied {
						b.Fatalf("response = %#v, want denied=%t", response, tc.wantDenied)
					}
				}
			})
		}
	}
}
