// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package breaktheglass

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	mc "github.com/projectcapsule/capsule/internal/mocks/client"
	"github.com/projectcapsule/capsule/internal/webhook/test"
	"github.com/projectcapsule/capsule/pkg/api/breaktheglass"
	gm "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ = Describe("BreakRequestTemplate Webhook", func() {
	var (
		brt       *capsulev1beta2.BreakRequestTemplate
		validator *breakRequestTemplateValidationHandler
		mockCtrl  *gm.Controller
		reader    *mc.MockReader
		cl        *mc.MockClient
		decoder   admission.Decoder
	)

	BeforeEach(func() {
		mockCtrl = gm.NewController(GinkgoT())
		reader = mc.NewMockReader(mockCtrl)
		cl = mc.NewMockClient(mockCtrl)
		brt = &capsulev1beta2.BreakRequestTemplate{
			Spec: capsulev1beta2.BreakRequestTemplateSpec{
				Items: breaktheglass.TemplateItems{
					"cm": breaktheglass.TemplateItem{
						ManifestTemplate: runtime.RawExtension{Object: &corev1.ConfigMap{}},
					},
				},
			},
		}
		decoder = &test.Decoder[*capsulev1beta2.BreakRequestTemplate]{
			Object: brt,
		}
		validator = &breakRequestTemplateValidationHandler{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(brt).NotTo(BeNil(), "Expected brt to be initialized")
	})
	AfterEach(func() {
		defer mockCtrl.Finish()
	})

	Context("When creating or updating BreakRequestTemplate under Validating Webhook", func() {
		Context("When auto approval is enabled an condition is not empty", func() {
			BeforeEach(func() {
				brt.Spec.AutoApprove = false
				brt.Spec.ApprovalCondition = "foo"
			})
			It("Should deny creation", func() {
				By("simulating an invalid creation scenario")

				response := validator.OnCreate(cl, reader, decoder, nil)(ctx, admission.Request{})
				test.VerifyResponse(response, http.StatusForbidden, "approvalCondition should not be set when autoApprove is false")
			})
			It("Should deny update", func() {
				By("simulating an invalid update scenario")

				response := validator.OnUpdate(cl, reader, decoder, nil)(ctx, admission.Request{})
				test.VerifyResponse(response, http.StatusForbidden, "approvalCondition should not be set when autoApprove is false")
			})
		})

		Context("When auto approval is enabled an condition is empty", func() {
			BeforeEach(func() {
				brt.Spec.AutoApprove = true
			})
			It("Should allow creation", func() {
				By("simulating an valid creation scenario")
				response := validator.OnCreate(cl, reader, decoder, nil)(ctx, admission.Request{})
				Expect(response).To(BeNil())
			})
			It("Should allow update", func() {
				By("simulating an valid update scenario")
				response := validator.OnUpdate(cl, reader, decoder, nil)(ctx, admission.Request{})
				Expect(response).To(BeNil())
			})
		})

		Context("When auto approval is enabled an condition is invalid", func() {
			BeforeEach(func() {
				brt.Spec.AutoApprove = true
				brt.Spec.ApprovalCondition = "foo.spec.reason == 'test'"
			})
			It("Should deny creation", func() {
				By("simulating an invalid creation scenario")
				response := validator.OnCreate(cl, reader, decoder, nil)(ctx, admission.Request{})
				test.VerifyResponse(response, http.StatusForbidden, "approvalCondition is invalid: ERROR: <input>:1:1: undeclared reference to 'foo'")
			})
			It("Should deny update", func() {
				By("simulating an invalid update scenario")
				response := validator.OnUpdate(cl, reader, decoder, nil)(ctx, admission.Request{})
				test.VerifyResponse(response, http.StatusForbidden, "approvalCondition is invalid: ERROR: <input>:1:1: undeclared reference to 'foo'")
			})
		})

		Context("When auto approval is enabled an condition is valid", func() {
			BeforeEach(func() {
				brt.Spec.AutoApprove = true
				brt.Spec.ApprovalCondition = "request.spec.reason == 'test'"
			})
			It("Should allow creation", func() {
				By("simulating an valid creation scenario")
				response := validator.OnCreate(cl, reader, decoder, nil)(ctx, admission.Request{})
				Expect(response).To(BeNil())
			})
			It("Should allow update", func() {
				By("simulating an valid update scenario")
				response := validator.OnUpdate(cl, reader, decoder, nil)(ctx, admission.Request{})
				Expect(response).To(BeNil())
			})
		})

		Context("When item schema is defined and valid", func() {
			BeforeEach(func() {
				brt.Spec.Items = breaktheglass.TemplateItems{
					"test": {
						ParamSchema: runtime.RawExtension{Raw: []byte(`{"type": "string"}`)},
					},
				}
			})
			It("Should allow creation", func() {
				By("simulating an valid creation scenario")
				response := validator.OnCreate(cl, reader, decoder, nil)(ctx, admission.Request{})
				Expect(response).To(BeNil())
			})
			It("Should allow update", func() {
				By("simulating an valid update scenario")
				response := validator.OnUpdate(cl, reader, decoder, nil)(ctx, admission.Request{})
				Expect(response).To(BeNil())
			})
		})
		Context("When item schema is defined and invalid", func() {
			BeforeEach(func() {
				brt.Spec.Items = breaktheglass.TemplateItems{
					"test": {
						ParamSchema: runtime.RawExtension{Raw: []byte(`"type": `)},
					},
				}
			})
			It("Should allow creation", func() {
				By("simulating an invalid creation scenario")
				response := validator.OnCreate(cl, reader, decoder, nil)(ctx, admission.Request{})
				test.VerifyResponse(response, http.StatusForbidden, `error rendering template: paramSchema for item "test" is invalid: failed to validate OpenAPI schemaData: schema invalid`)
			})
			It("Should allow update", func() {
				By("simulating an invalid update scenario")
				response := validator.OnUpdate(cl, reader, decoder, nil)(ctx, admission.Request{})
				test.VerifyResponse(response, http.StatusForbidden, `error rendering template: paramSchema for item "test" is invalid: failed to validate OpenAPI schemaData: schema invalid`)
			})
		})
	})
})
