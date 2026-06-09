// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package breaktheglass

import (
	"errors"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	mc "github.com/projectcapsule/capsule/internal/mocks/client"
	"github.com/projectcapsule/capsule/internal/webhook/test"
	gm "go.uber.org/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	defaultTemplateName   = "foo"
	alternateTemplateName = "bar"
)

var _ = Describe("BreakRequest Webhook", func() {
	var (
		br        *capsulev1beta2.BreakRequest
		validator *breakRequestValidationHandler
		mockCtrl  *gm.Controller
		reader    *mc.MockReader
		decoder   admission.Decoder
	)

	BeforeEach(func() {
		mockCtrl = gm.NewController(GinkgoT())
		reader = mc.NewMockReader(mockCtrl)
		br = &capsulev1beta2.BreakRequest{}
		decoder = &test.Decoder[*capsulev1beta2.BreakRequest]{
			Object: br,
		}
		validator = &breakRequestValidationHandler{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(br).NotTo(BeNil(), "Expected obj to be initialized")
	})
	AfterEach(func() {
		defer mockCtrl.Finish()
	})

	Context("When creating BreakRequest under Validating Webhook", func() {
		It("Should deny creation if the referenced template is not available", func() {
			By("simulating an invalid creation scenario")
			br.Spec.TemplateName = defaultTemplateName
			reader.EXPECT().
				Get(gm.Any(), client.ObjectKey{Name: br.Spec.TemplateName}, gm.Any(), gm.Any()).
				Return(errors.New("not found"))
			response := validator.OnCreate(nil, reader, decoder, nil)(ctx, admission.Request{})
			test.VerifyResponse(response, http.StatusInternalServerError, "error loading template foo: not found")
		})
		It("Should allow creation if the referenced template is available", func() {
			By("simulating an invalid creation scenario")
			br.Spec.TemplateName = "bar"
			reader.EXPECT().
				Get(gm.Any(), client.ObjectKey{Name: br.Spec.TemplateName}, gm.Any(), gm.Any())
			response := validator.OnCreate(nil, reader, decoder, nil)(ctx, admission.Request{})
			Expect(response).To(BeNil())
		})
		It("Should deny creation if duration exceeds the template max duration", func() {
			br.Spec.Duration.Duration = time.Hour
			reader.EXPECT().
				Get(gm.Any(), client.ObjectKey{Name: br.Spec.TemplateName}, gm.Any(), gm.Any()).
				Do(func(_ any, _ any, brt *capsulev1beta2.BreakRequestTemplate, _ ...any) {
					brt.Spec.MaxDuration.Duration = time.Minute
				})

			response := validator.OnCreate(nil, reader, decoder, nil)(ctx, admission.Request{})
			test.VerifyResponse(response, http.StatusForbidden, "requested duration 1h0m0s exceeds template maxDuration 1m0s")
		})
	})

	Context("When updating BreakRequest under Validating Webhook", func() {
		var oldBr *capsulev1beta2.BreakRequest

		BeforeEach(func() {
			oldBr = &capsulev1beta2.BreakRequest{}
			decoder = &test.Decoder[*capsulev1beta2.BreakRequest]{
				Object:    br,
				OldObject: oldBr,
			}
		})
		It("Should be valid if the template name is not changed", func() {
			oldBr.Spec.TemplateName = defaultTemplateName
			br.Spec.TemplateName = defaultTemplateName
			response := validator.OnUpdate(nil, reader, decoder, nil)(ctx, admission.Request{})
			Expect(response).To(BeNil())
		})
		It("Should not be allowed to change the templateName", func() {
			oldBr.Spec.TemplateName = defaultTemplateName
			br.Spec.TemplateName = alternateTemplateName

			response := validator.OnUpdate(nil, reader, decoder, nil)(ctx, admission.Request{})
			test.VerifyResponse(response, http.StatusForbidden, "templateName cannot be changed. old: foo, new: bar")
		})
	})
})
