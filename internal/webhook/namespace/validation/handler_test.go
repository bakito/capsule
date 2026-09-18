// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package validation

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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/pkg/api/meta"
	"github.com/projectcapsule/capsule/pkg/api/rbac"
	ad "github.com/projectcapsule/capsule/pkg/runtime/admission"
	"github.com/projectcapsule/capsule/pkg/runtime/configuration"
	"github.com/projectcapsule/capsule/pkg/runtime/events"
	"github.com/projectcapsule/capsule/pkg/runtime/handlers"
	"github.com/projectcapsule/capsule/pkg/users"
)

func TestNamespaceHandlerAllowsUnchangedFinalizeWithoutTenant(t *testing.T) {
	t.Parallel()

	scheme := namespaceValidationScheme(t)
	now := metav1.Now()
	oldNs := namespaceWithTenantReference("workloads", "missing", "missing-uid")
	oldNs.DeletionTimestamp = &now
	oldNs.Status.Phase = corev1.NamespaceTerminating
	newNs := oldNs.DeepCopy()
	newNs.Spec.Finalizers = nil

	response := NamespaceHandler(nil).OnUpdate(
		nil,
		nil,
		admission.NewDecoder(scheme),
		nil,
	)(context.Background(), namespaceUpdateRequest(t, oldNs, newNs, "finalize"))

	if response != nil {
		t.Fatalf("finalize response = %#v, want no interception", response)
	}
}

func TestNamespaceHandlerRejectsTenantChangeDuringFinalize(t *testing.T) {
	t.Parallel()

	scheme := namespaceValidationScheme(t)
	now := metav1.Now()
	oldNs := namespaceWithTenantReference("workloads", "solar", "solar-uid")
	oldNs.DeletionTimestamp = &now
	oldNs.Status.Phase = corev1.NamespaceTerminating
	newNs := namespaceWithTenantReference("workloads", "lunar", "lunar-uid")
	newNs.DeletionTimestamp = &now
	newNs.Status.Phase = corev1.NamespaceTerminating

	response := NamespaceHandler(nil).OnUpdate(
		nil,
		nil,
		admission.NewDecoder(scheme),
		nil,
	)(context.Background(), namespaceUpdateRequest(t, oldNs, newNs, "finalize"))

	if response == nil || response.Allowed {
		t.Fatalf("finalize response = %#v, want tenant assignment denial", response)
	}
}

func TestNamespaceHandlerAllowsDeleteWithMissingTenant(t *testing.T) {
	t.Parallel()

	scheme := namespaceValidationScheme(t)
	reader := fake.NewClientBuilder().WithScheme(scheme).Build()
	oldNs := namespaceWithTenantReference("workloads", "missing", "missing-uid")
	raw, err := json.Marshal(oldNs)
	if err != nil {
		t.Fatal(err)
	}

	response := NamespaceHandler(nil).OnDelete(
		reader,
		reader,
		admission.NewDecoder(scheme),
		nil,
	)(context.Background(), admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		Operation: admissionv1.Delete,
		OldObject: runtime.RawExtension{Raw: raw},
	}})

	if response != nil {
		t.Fatalf("delete response = %#v, want missing Tenant to be ignored", response)
	}
}

type recordingNamespaceHandler struct {
	updates  int
	response *admission.Response
}

func (h *recordingNamespaceHandler) OnCreate(
	client.Client,
	client.Reader,
	users.AdmissionUser,
	*corev1.Namespace,
	admission.Decoder,
	events.EventRecorder,
	*capsulev1beta2.Tenant,
) handlers.Func {
	return func(context.Context, admission.Request) *admission.Response { return nil }
}

func (h *recordingNamespaceHandler) OnUpdate(
	client.Client,
	client.Reader,
	users.AdmissionUser,
	*corev1.Namespace,
	*corev1.Namespace,
	admission.Decoder,
	events.EventRecorder,
	*capsulev1beta2.Tenant,
) handlers.Func {
	return func(context.Context, admission.Request) *admission.Response {
		h.updates++

		return h.response
	}
}

func (h *recordingNamespaceHandler) OnDelete(
	client.Client,
	client.Reader,
	users.AdmissionUser,
	*corev1.Namespace,
	admission.Decoder,
	events.EventRecorder,
	*capsulev1beta2.Tenant,
) handlers.Func {
	return func(context.Context, admission.Request) *admission.Response { return nil }
}

// TestNamespaceHandlerSubresourceMetadataWrites covers namespace metadata
// writes carried by the namespaces/status and namespaces/finalize
// subresources against live (non-terminating) namespaces.
func TestNamespaceHandlerSubresourceMetadataWrites(t *testing.T) {
	t.Parallel()

	const (
		ownerName    = "alice"
		strangerName = "bob"
		outsiderName = "mallory"
	)

	solarOwner := rbac.CoreOwnerSpec{UserSpec: rbac.UserSpec{Name: ownerName, Kind: rbac.UserOwner}}
	lunarOwner := rbac.CoreOwnerSpec{UserSpec: rbac.UserSpec{Name: strangerName, Kind: rbac.UserOwner}}

	solar := &capsulev1beta2.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "solar", UID: "solar-uid"}}
	solar.Status.Owners = rbac.OwnerStatusListSpec{solarOwner}
	lunar := &capsulev1beta2.Tenant{ObjectMeta: metav1.ObjectMeta{Name: "lunar", UID: "lunar-uid"}}
	lunar.Status.Owners = rbac.OwnerStatusListSpec{lunarOwner}

	configurationObject := &capsulev1beta2.CapsuleConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "capsule"},
		Status: capsulev1beta2.CapsuleConfigurationStatus{
			Users: rbac.UserListSpec{solarOwner.UserSpec, lunarOwner.UserSpec},
		},
	}

	managed := namespaceWithTenantReference("solar-prod", solar.Name, string(solar.UID))
	unmanaged := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}}

	withLabel := func(ns *corev1.Namespace) *corev1.Namespace {
		out := ns.DeepCopy()
		if out.Labels == nil {
			out.Labels = map[string]string{}
		}

		out.Labels["pod-security.kubernetes.io/enforce"] = "privileged"

		return out
	}

	terminating := managed.DeepCopy()
	now := metav1.Now()
	terminating.DeletionTimestamp = &now
	terminating.Status.Phase = corev1.NamespaceTerminating

	tests := []struct {
		name         string
		user         string
		oldNs, newNs *corev1.Namespace
		subresource  string
		subHandler   *admission.Response
		wantDenial   string
		wantHandlers int
	}{
		{
			name:         "owner finalize on own namespace reaches metadata handlers",
			user:         ownerName,
			oldNs:        managed,
			newNs:        withLabel(managed),
			subresource:  "finalize",
			wantHandlers: 1,
		},
		{
			name:         "owner status on own namespace reaches metadata handlers",
			user:         ownerName,
			oldNs:        managed,
			newNs:        withLabel(managed),
			subresource:  "status",
			wantHandlers: 1,
		},
		{
			name:         "owner finalize propagates metadata handler denial",
			user:         ownerName,
			oldNs:        managed,
			newNs:        withLabel(managed),
			subresource:  "finalize",
			subHandler:   ad.Deny("label is forbidden for the current Tenant"),
			wantDenial:   "label is forbidden for the current Tenant",
			wantHandlers: 1,
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
			name:        "tenant user plain update on unmanaged namespace is denied",
			user:        ownerName,
			oldNs:       unmanaged,
			newNs:       withLabel(unmanaged),
			wantDenial:  "namespace is not owned by any tenant",
		},
		{
			name:        "unrelated user status on unmanaged namespace is not intercepted",
			user:        outsiderName,
			oldNs:       unmanaged,
			newNs:       withLabel(unmanaged),
			subresource: "status",
		},
		{
			name:        "owner finalize on terminating namespace skips metadata handlers",
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
			scheme := namespaceValidationScheme(t)
			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(configurationObject.DeepCopy(), solar.DeepCopy(), lunar.DeepCopy()).
				Build()
			cfg := configuration.NewCapsuleConfiguration(ctx, cl, cl, nil, configurationObject.Name)
			recorder := events.NewEventRecorder(nil, logr.Discard(), nil, nil)
			sub := &recordingNamespaceHandler{response: tt.subHandler}

			req := namespaceUpdateRequest(t, tt.oldNs, tt.newNs, tt.subresource)
			req.UserInfo = authenticationv1.UserInfo{Username: tt.user}

			response := NamespaceHandler(cfg, sub).OnUpdate(
				cl,
				cl,
				admission.NewDecoder(scheme),
				recorder,
			)(ctx, req)

			if sub.updates != tt.wantHandlers {
				t.Fatalf("metadata handler invocations = %d, want %d", sub.updates, tt.wantHandlers)
			}

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

func namespaceValidationScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := capsulev1beta2.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	return scheme
}

func namespaceWithTenantReference(name, tenantName, tenantUID string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   name,
		Labels: map[string]string{meta.TenantLabel: tenantName},
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: capsulev1beta2.GroupVersion.String(),
			Kind:       "Tenant",
			Name:       tenantName,
			UID:        types.UID(tenantUID),
		}},
	}}
}

func namespaceUpdateRequest(
	t *testing.T,
	oldNs, newNs *corev1.Namespace,
	subresource string,
) admission.Request {
	t.Helper()

	oldRaw, err := json.Marshal(oldNs)
	if err != nil {
		t.Fatal(err)
	}
	newRaw, err := json.Marshal(newNs)
	if err != nil {
		t.Fatal(err)
	}

	return admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		Operation:   admissionv1.Update,
		SubResource: subresource,
		Object:      runtime.RawExtension{Raw: newRaw},
		OldObject:   runtime.RawExtension{Raw: oldRaw},
	}}
}
