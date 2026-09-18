// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package mutation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/pkg/api/meta"
	"github.com/projectcapsule/capsule/pkg/api/rbac"
	"github.com/projectcapsule/capsule/pkg/runtime/configuration"
	capevents "github.com/projectcapsule/capsule/pkg/runtime/events"
)

func TestNamespaceHandlerDoesNotInterceptUnlabelledAdministratorCreate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	if err := capsulev1beta2.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	const (
		configurationName = "capsule"
		administratorName = "configured-administrator"
	)

	configurationObject := &capsulev1beta2.CapsuleConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: configurationName},
		Spec: capsulev1beta2.CapsuleConfigurationSpec{
			Administrators: rbac.UserListSpec{{
				Name: administratorName,
				Kind: rbac.UserOwner,
			}},
		},
	}
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configurationObject).
		Build()
	cfg := configuration.NewCapsuleConfiguration(ctx, cl, cl, nil, configurationName)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   "unassigned",
		Labels: map[string]string{"example.com/label": "value"},
	}}
	raw, err := json.Marshal(ns)
	if err != nil {
		t.Fatal(err)
	}

	response := NamespaceHandler(cfg).OnCreate(
		cl,
		cl,
		admission.NewDecoder(scheme),
		nil,
	)(ctx, admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		Object: runtime.RawExtension{Raw: raw},
		UserInfo: authenticationv1.UserInfo{
			Username: administratorName,
		},
	}})

	if response != nil {
		t.Fatalf("expected unlabelled administrator create not to be intercepted, got %#v", response)
	}
}

func TestNamespaceHandlerDoesNotInterceptFinalize(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	now := metav1.Now()
	oldNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "terminating",
			DeletionTimestamp: &now,
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating},
	}
	newNs := oldNs.DeepCopy()
	newNs.Spec.Finalizers = nil

	oldRaw, err := json.Marshal(oldNs)
	if err != nil {
		t.Fatal(err)
	}
	newRaw, err := json.Marshal(newNs)
	if err != nil {
		t.Fatal(err)
	}

	response := NamespaceHandler(nil).OnUpdate(
		nil,
		nil,
		admission.NewDecoder(scheme),
		nil,
	)(context.Background(), admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		Operation:   admissionv1.Update,
		SubResource: "finalize",
		Object:      runtime.RawExtension{Raw: newRaw},
		OldObject:   runtime.RawExtension{Raw: oldRaw},
	}})

	if response != nil {
		t.Fatalf("finalize response = %#v, want no interception", response)
	}
}

// TestNamespaceHandlerGuardsSubresourceWritesOnLiveNamespaces proves the
// ownership gate of the mutating webhook sees namespaces/status and
// namespaces/finalize writes against live namespaces exactly like plain
// namespace updates, while genuinely terminating namespaces are skipped.
func TestNamespaceHandlerGuardsSubresourceWritesOnLiveNamespaces(t *testing.T) {
	t.Parallel()

	const (
		ownerName    = "alice"
		strangerName = "bob"
	)

	owner := rbac.CoreOwnerSpec{UserSpec: rbac.UserSpec{Name: ownerName, Kind: rbac.UserOwner}}
	stranger := rbac.CoreOwnerSpec{UserSpec: rbac.UserSpec{Name: strangerName, Kind: rbac.UserOwner}}
	green := testTenant("green", "green-uid")
	green.Status.Owners = rbac.OwnerStatusListSpec{owner}
	blue := testTenant("blue", "blue-uid")
	blue.Status.Owners = rbac.OwnerStatusListSpec{stranger}
	configurationObject := &capsulev1beta2.CapsuleConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "capsule"},
		Status: capsulev1beta2.CapsuleConfigurationStatus{
			Users: rbac.UserListSpec{owner.UserSpec, stranger.UserSpec},
		},
	}

	managed := testTenantNamespace("workloads", green)
	unmanaged := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}

	withLabel := func(ns *corev1.Namespace) *corev1.Namespace {
		out := ns.DeepCopy()
		if out.Labels == nil {
			out.Labels = map[string]string{}
		}

		out.Labels["pod-security.kubernetes.io/enforce"] = "privileged"

		return out
	}

	now := metav1.Now()
	terminating := unmanaged.DeepCopy()
	terminating.DeletionTimestamp = &now
	terminating.Status.Phase = corev1.NamespaceTerminating

	tests := []struct {
		name         string
		user         string
		oldNs, newNs *corev1.Namespace
		subresource  string
		wantDenial   string
	}{
		{
			name:        "owner status on own namespace is allowed",
			user:        ownerName,
			oldNs:       managed,
			newNs:       withLabel(managed),
			subresource: "status",
		},
		{
			name:        "owner finalize on own namespace is allowed",
			user:        ownerName,
			oldNs:       managed,
			newNs:       withLabel(managed),
			subresource: "finalize",
		},
		{
			name:        "other tenant owner finalize on foreign namespace is denied",
			user:        strangerName,
			oldNs:       managed,
			newNs:       withLabel(managed),
			subresource: "finalize",
			wantDenial:  "denied patch request for this namespace",
		},
		{
			name:        "tenant user status on unmanaged namespace is denied",
			user:        ownerName,
			oldNs:       unmanaged,
			newNs:       withLabel(unmanaged),
			subresource: "status",
			wantDenial:  "namespace is not owned by any tenant",
		},
		{
			name:        "tenant user finalize on unmanaged namespace is denied",
			user:        ownerName,
			oldNs:       unmanaged,
			newNs:       withLabel(unmanaged),
			subresource: "finalize",
			wantDenial:  "namespace is not owned by any tenant",
		},
		{
			name:        "tenant user finalize on terminating namespace is not intercepted",
			user:        ownerName,
			oldNs:       terminating,
			newNs:       withLabel(terminating),
			subresource: "finalize",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			scheme := testScheme(t)
			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(configurationObject.DeepCopy(), green.DeepCopy(), blue.DeepCopy()).
				Build()
			cfg := configuration.NewCapsuleConfiguration(ctx, cl, cl, nil, configurationObject.Name)
			recorder := capevents.NewEventRecorder(nil, logr.Discard(), nil, nil)

			oldRaw, err := json.Marshal(tt.oldNs)
			if err != nil {
				t.Fatal(err)
			}
			newRaw, err := json.Marshal(tt.newNs)
			if err != nil {
				t.Fatal(err)
			}

			response := NamespaceHandler(cfg, OwnerReferenceHandler(cfg)).OnUpdate(
				cl,
				cl,
				admission.NewDecoder(scheme),
				recorder,
			)(ctx, admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
				Operation:   admissionv1.Update,
				SubResource: tt.subresource,
				Object:      runtime.RawExtension{Raw: newRaw},
				OldObject:   runtime.RawExtension{Raw: oldRaw},
				UserInfo:    authenticationv1.UserInfo{Username: tt.user},
			}})

			if tt.wantDenial == "" {
				if response != nil && !response.Allowed {
					t.Fatalf("response = %#v, want allow", response)
				}

				return
			}

			if response == nil || response.Allowed {
				t.Fatalf("response = %#v, want denial %q", response, tt.wantDenial)
			}

			if !strings.Contains(response.Result.Message, tt.wantDenial) {
				t.Fatalf("denial message = %q, want %q", response.Result.Message, tt.wantDenial)
			}
		})
	}
}

func TestNamespaceHandlerRejectsTenantOwnerLabelMigrationWithEmptyOwnerReferences(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scheme := testScheme(t)
	owner := rbac.CoreOwnerSpec{UserSpec: rbac.UserSpec{Name: "alice", Kind: rbac.UserOwner}}
	green := testTenant("green", "green-uid")
	green.Status.Owners = rbac.OwnerStatusListSpec{owner}
	blue := testTenant("blue", "blue-uid")
	configurationObject := &capsulev1beta2.CapsuleConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "capsule"},
		Status: capsulev1beta2.CapsuleConfigurationStatus{
			Users: rbac.UserListSpec{owner.UserSpec},
		},
	}
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(configurationObject, green, blue).
		Build()
	cfg := configuration.NewCapsuleConfiguration(ctx, cl, cl, nil, configurationObject.Name)
	recorder := capevents.NewEventRecorder(nil, logr.Discard(), nil, nil)

	oldNs := testTenantNamespace("workloads", green)
	newNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   oldNs.Name,
		Labels: map[string]string{meta.TenantLabel: blue.Name},
	}}
	oldRaw, err := json.Marshal(oldNs)
	if err != nil {
		t.Fatal(err)
	}
	newRaw, err := json.Marshal(newNs)
	if err != nil {
		t.Fatal(err)
	}

	response := NamespaceHandler(cfg, OwnerReferenceHandler(cfg)).OnUpdate(
		cl,
		cl,
		admission.NewDecoder(scheme),
		recorder,
	)(ctx, admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		Object:    runtime.RawExtension{Raw: newRaw},
		OldObject: runtime.RawExtension{Raw: oldRaw},
		UserInfo: authenticationv1.UserInfo{
			Username: owner.Name,
		},
	}})

	if response == nil || response.Allowed {
		t.Fatalf("expected label migration patch to be denied, got %#v", response)
	}
}
