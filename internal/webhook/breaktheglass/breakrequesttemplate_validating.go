// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package breaktheglass

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/capsule/internal/breaktheglass/conditions"
	"github.com/projectcapsule/capsule/internal/breaktheglass/template"
	ad "github.com/projectcapsule/capsule/pkg/runtime/admission"
	"github.com/projectcapsule/capsule/pkg/runtime/handlers"
)

func BreakRequestTemplateValidationHandler(log logr.Logger) handlers.Handler {
	return &breakRequestTemplateValidationHandler{
		log: log,
	}
}

type breakRequestTemplateValidationHandler struct {
	log logr.Logger
}

func (b *breakRequestTemplateValidationHandler) OnCreate(cl client.Client, _ client.Reader, decoder admission.Decoder, _ events.EventRecorder) handlers.Func {
	return func(ctx context.Context, req admission.Request) *admission.Response {
		b.log.Info("Validation for BreakRequestTemplate upon update", "name", req.Name)

		return validate(ctx, cl, decoder, req)
	}
}

func (b *breakRequestTemplateValidationHandler) OnDelete(_ client.Client, _ client.Reader, _ admission.Decoder, _ events.EventRecorder) handlers.Func {
	return func(_ context.Context, _ admission.Request) *admission.Response {
		return nil
	}
}

func (b *breakRequestTemplateValidationHandler) OnUpdate(cl client.Client, _ client.Reader, decoder admission.Decoder, _ events.EventRecorder) handlers.Func {
	return func(ctx context.Context, req admission.Request) *admission.Response {
		b.log.Info("Validation for BreakRequestTemplate upon update", "name", req.Name)

		return validate(ctx, cl, decoder, req)
	}
}

func validate(ctx context.Context, c client.Client, decoder admission.Decoder, req admission.Request) *admission.Response {
	brt := &capsulev1beta2.BreakRequestTemplate{}
	if err := decoder.Decode(req, brt); err != nil {
		return ad.Denyf("failed to decode new object: %v", err)
	}

	if len(brt.Spec.Items) == 0 {
		return ad.Deny("items must be set")
	}

	for _, item := range brt.Spec.Items {
		if item.ManifestTemplate.Object == nil && len(item.ManifestTemplate.Raw) == 0 {
			return ad.Deny("manifestTemplate must be set")
		}
	}

	if !brt.Spec.AutoApprove {
		if brt.Spec.ApprovalCondition != "" {
			return ad.Denyf("approvalCondition should not be set when autoApprove is false")
		}
	} else {
		if brt.Spec.ApprovalCondition == "" {
			return nil
		}

		if _, err := conditions.PrepareCondition(brt); err != nil {
			return ad.Denyf("approvalCondition is invalid: %v", err)
		}
	}

	if err := template.ValidateItems(brt.Spec.Items); err != nil {
		return ad.Denyf("error rendering template: %v", err)
	}

	if err := validateControllerCanManageTemplateItems(ctx, c, brt); err != nil {
		return ad.Deny(err.Error())
	}

	return nil
}

func validateControllerCanManageTemplateItems(ctx context.Context, c client.Client, brt *capsulev1beta2.BreakRequestTemplate) error {
	crudVerbs := []string{"create", "get", "list", "watch", "update", "patch", "delete"}

	ras, err := getResourceAttributes(brt, c.RESTMapper())
	if err != nil {
		return fmt.Errorf("cannot prepare admission review attributes: %w", err)
	}

	for _, ra := range ras {
		for _, verb := range crudVerbs {
			verbRa := ra.DeepCopy()
			verbRa.Verb = verb
			verbRa.Namespace = ""
			review := &authorizationv1.SelfSubjectAccessReview{
				Spec: authorizationv1.SelfSubjectAccessReviewSpec{
					ResourceAttributes: verbRa,
				},
			}

			if err := c.Create(ctx, review); err != nil {
				if apierrors.IsForbidden(err) {
					return fmt.Errorf("controller is not allowed to review permissions for manifestTemplate resource %q", gvk(verbRa))
				}

				return fmt.Errorf("failed to review controller permissions for manifestTemplate resource %q and verb %q: %w", gvk(verbRa), verb, err)
			}

			if !review.Status.Allowed {
				reason := review.Status.Reason
				if reason == "" {
					reason = "permission denied"
				}

				return fmt.Errorf(
					"controller is not allowed to manage manifestTemplate: cannot %s %s: %s",
					verb,
					gvk(verbRa),
					reason,
				)
			}
		}
	}

	return nil
}

func gvk(ra *authorizationv1.ResourceAttributes) string {
	return fmt.Sprintf("%s.%s/%s", ra.Resource, ra.Version, ra.Group)
}

func getResourceAttributes(brt *capsulev1beta2.BreakRequestTemplate, restMapper k8smeta.RESTMapper) ([]authorizationv1.ResourceAttributes, error) {
	seen := make(map[string]bool)
	attributes := make([]authorizationv1.ResourceAttributes, 0)

	for itemName, item := range brt.Spec.Items {
		var gvk schema.GroupVersionKind
		if item.ManifestTemplate.Object != nil {
			gvk = item.ManifestTemplate.Object.GetObjectKind().GroupVersionKind()
		} else {
			obj := &unstructured.Unstructured{}
			if _, _, err := unstructured.UnstructuredJSONScheme.Decode(item.ManifestTemplate.Raw, nil, obj); err != nil {
				return nil, fmt.Errorf("manifestTemplate for item %q is invalid: %w", itemName, err)
			}

			gvk = obj.GroupVersionKind()
			if gvk.Empty() {
				return nil, fmt.Errorf("manifestTemplate for item %q must define apiVersion and kind", itemName)
			}
		}

		mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve resource for manifestTemplate item %q with kind %q: %w", itemName, gvk.String(), err)
		}

		attr := authorizationv1.ResourceAttributes{
			Group:    mapping.Resource.Group,
			Version:  mapping.Resource.Version,
			Resource: mapping.Resource.Resource,
		}

		key := fmt.Sprintf(
			"%s/%s/%s",
			attr.Group,
			attr.Version,
			attr.Resource,
		)

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = true

		attributes = append(attributes, attr)
	}

	return attributes, nil
}
