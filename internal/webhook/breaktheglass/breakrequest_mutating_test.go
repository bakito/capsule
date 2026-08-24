// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package breaktheglass

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrl "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/internal/webhook/test"
	"github.com/projectcapsule/capsule/pkg/api/rbac"
	"github.com/projectcapsule/capsule/pkg/runtime/configuration"
)

type mockConfig struct {
	configuration.Configuration
}

func (m mockConfig) IgnoreUserWithGroups() []string   { return nil }
func (m mockConfig) Administrators() rbac.UserListSpec { return nil }
func (m mockConfig) GetUsersByStatus() rbac.UserListSpec { return nil }

func TestBreakRequestMutationHandler(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = capsulev1beta2.AddToScheme(scheme)
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()

	ctx := context.Background()
	log := ctrl.Log.WithName("test")
	cfg := mockConfig{}
	handler := BreakRequestMutationHandler(cfg, log)

	t.Run("OnCreate", func(t *testing.T) {
		br := &capsulev1beta2.BreakRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-request",
			},
		}
		decoder := &test.Decoder[*capsulev1beta2.BreakRequest]{
			Object: br,
		}
		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UserInfo: authenticationv1.UserInfo{
					Username: "alice",
					Groups:   []string{"dev"},
				},
				Object: runtime.RawExtension{
					Raw: func() []byte {
						b, _ := json.Marshal(br)
						return b
					}(),
				},
			},
		}

		resp := handler.OnCreate(cl, nil, decoder, nil)(ctx, req)
		assert.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Patches)
	})

	t.Run("OnUpdate", func(t *testing.T) {
		t.Run("transition to approved", func(t *testing.T) {
			oldBr := &capsulev1beta2.BreakRequest{
				Status: capsulev1beta2.BreakRequestStatus{
					Phase: capsulev1beta2.RequestPhaseRequested,
				},
			}
			newBr := &capsulev1beta2.BreakRequest{
				Status: capsulev1beta2.BreakRequestStatus{
					Phase: capsulev1beta2.RequestPhaseApproved,
				},
			}
			decoder := &test.Decoder[*capsulev1beta2.BreakRequest]{
				Object:    newBr,
				OldObject: oldBr,
			}
			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					UserInfo: authenticationv1.UserInfo{
						Username: "admin",
					},
					Object: runtime.RawExtension{
						Raw: func() []byte {
							b, _ := json.Marshal(newBr)
							return b
						}(),
					},
				},
			}

			resp := handler.OnUpdate(cl, nil, decoder, nil)(ctx, req)
			assert.NotNil(t, resp)
			assert.True(t, resp.Allowed)
			assert.NotEmpty(t, resp.Patches)
		})

		t.Run("no transition", func(t *testing.T) {
			oldBr := &capsulev1beta2.BreakRequest{
				Status: capsulev1beta2.BreakRequestStatus{
					Phase: capsulev1beta2.RequestPhaseApproved,
					Review: &capsulev1beta2.ReviewInfo{
						ReviewerInfo: &capsulev1beta2.BreakRequestUserInfo{
							Username: "original-reviewer",
						},
					},
				},
			}
			newBr := &capsulev1beta2.BreakRequest{
				Status: capsulev1beta2.BreakRequestStatus{
					Phase: capsulev1beta2.RequestPhaseApproved,
				},
			}
			decoder := &test.Decoder[*capsulev1beta2.BreakRequest]{
				Object:    newBr,
				OldObject: oldBr,
			}
			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					UserInfo: authenticationv1.UserInfo{
						Username: "someone-else",
					},
					Object: runtime.RawExtension{
						Raw: func() []byte {
							b, _ := json.Marshal(newBr)
							return b
						}(),
					},
				},
			}

			resp := handler.OnUpdate(cl, nil, decoder, nil)(ctx, req)
			assert.NotNil(t, resp)
			assert.True(t, resp.Allowed)
			assert.Empty(t, resp.Patches)
		})
	})

	t.Run("OnDelete", func(t *testing.T) {
		resp := handler.OnDelete(cl, nil, nil, nil)(ctx, admission.Request{})
		assert.Nil(t, resp)
	})

	t.Run("ServiceAccount resolving", func(t *testing.T) {
		br := &capsulev1beta2.BreakRequest{}
		decoder := &test.Decoder[*capsulev1beta2.BreakRequest]{
			Object: br,
		}
		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UserInfo: authenticationv1.UserInfo{
					Username: "system:serviceaccount:capsule-system:capsule-controller",
				},
				Object: runtime.RawExtension{
					Raw: func() []byte {
						b, _ := json.Marshal(br)
						return b
					}(),
				},
			},
		}

		resp := handler.OnCreate(cl, nil, decoder, nil)(ctx, req)
		assert.NotNil(t, resp)
		assert.True(t, resp.Allowed)
		assert.NotEmpty(t, resp.Patches)
	})
}
