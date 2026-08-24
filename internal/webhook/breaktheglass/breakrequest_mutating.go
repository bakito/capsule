// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package breaktheglass

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	ad "github.com/projectcapsule/capsule/pkg/runtime/admission"
	"github.com/projectcapsule/capsule/pkg/runtime/configuration"
	"github.com/projectcapsule/capsule/pkg/users"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/pkg/runtime/events"
	"github.com/projectcapsule/capsule/pkg/runtime/handlers"
)

func BreakRequestMutationHandler(cfg configuration.Configuration, log logr.Logger) handlers.Handler {
	return &breakRequestMutationHandler{
		cfg: cfg,
		log: log,
	}
}

type breakRequestMutationHandler struct {
	cfg configuration.Configuration
	log logr.Logger
}

func (b *breakRequestMutationHandler) OnCreate(c client.Client, _ client.Reader, decoder admission.Decoder, _ events.EventRecorder) handlers.Func {
	return func(ctx context.Context, req admission.Request) *admission.Response {
		br := &capsulev1beta2.BreakRequest{}
		if err := decoder.Decode(req, br); err != nil {
			return ad.ErroredResponse(fmt.Errorf("failed to decode new object: %w", err))
		}
		user := handlers.ResolveAdmissionUser(ctx, c, req, b.cfg)

		brUser := asBreakRequestUser(user)

		br.Status.RequestedBy = brUser

		b.setReviewer(br, nil, brUser)

		marshaled, err := json.Marshal(br)
		if err != nil {
			return ad.ErroredResponse(err)
		}

		response := admission.PatchResponseFromRaw(req.Object.Raw, marshaled)

		return &response
	}
}

func (b *breakRequestMutationHandler) OnUpdate(c client.Client, _ client.Reader, decoder admission.Decoder, _ events.EventRecorder) handlers.Func {
	return func(ctx context.Context, req admission.Request) *admission.Response {
		br := &capsulev1beta2.BreakRequest{}
		if err := decoder.Decode(req, br); err != nil {
			return ad.ErroredResponse(fmt.Errorf("failed to decode new object: %w", err))
		}
		oldBr := &capsulev1beta2.BreakRequest{}
		if err := decoder.DecodeRaw(req.OldObject, oldBr); err != nil {
		}
		user := handlers.ResolveAdmissionUser(ctx, c, req, b.cfg)

		b.setReviewer(br, oldBr, asBreakRequestUser(user))

		marshaled, err := json.Marshal(br)
		if err != nil {
			return ad.ErroredResponse(err)
		}

		response := admission.PatchResponseFromRaw(req.Object.Raw, marshaled)

		return &response
	}
}

func (b *breakRequestMutationHandler) OnDelete(_ client.Client, _ client.Reader, _ admission.Decoder, _ events.EventRecorder) handlers.Func {
	return func(_ context.Context, _ admission.Request) *admission.Response {
		return nil
	}
}

func (b *breakRequestMutationHandler) setReviewer(br *capsulev1beta2.BreakRequest, oldBr *capsulev1beta2.BreakRequest, brUser *capsulev1beta2.BreakRequestUserInfo) {
	phase := br.Status.Phase

	// Set reviewer on create/update or when phase transitions to approved/denied
	isNewRequest := phase == "" || phase == capsulev1beta2.RequestPhaseRequested || oldBr == nil
	isApprovalTransition := phase == capsulev1beta2.RequestPhaseApproved && oldBr.Status.Phase != capsulev1beta2.RequestPhaseApproved
	isDenialTransition := phase == capsulev1beta2.RequestPhaseDenied && oldBr.Status.Phase != capsulev1beta2.RequestPhaseDenied

	if isNewRequest || isApprovalTransition || isDenialTransition {
		// set the current user as the reviewer until the request is reviewed
		if br.Status.Review == nil {
			br.Status.Review = &capsulev1beta2.ReviewInfo{}
		}

		br.Status.Review.ReviewerInfo = brUser
	}
}

func asBreakRequestUser(user users.AdmissionUser) *capsulev1beta2.BreakRequestUserInfo {
	return &capsulev1beta2.BreakRequestUserInfo{
		Username:       user.Username,
		Groups:         user.Groups,
		ServiceAccount: asServiceAccount(user.ServiceAccount),
	}
}

func asServiceAccount(account *users.AdmissionServiceAccount) *capsulev1beta2.BreakRequestServiceAccount {
	if account == nil {
		return nil
	}
	return &capsulev1beta2.BreakRequestServiceAccount{
		Namespace: account.Namespace,
		Name:      account.Name,
	}
}
