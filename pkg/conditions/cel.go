// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package conditions

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"k8s.io/apimachinery/pkg/runtime"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
)

// IsApproved evaluates whether a BreakRequest should be automatically approved based on the
// BreakRequestTemplate's auto-approval configuration and optional approval condition.
//
// Parameters:
//   - brt: The BreakRequestTemplate containing the auto-approval configuration and CEL condition
//   - br: The BreakRequest to evaluate for approval
//
// Returns:
//   - approved: true if the request is approved, false otherwise
//   - err: any error that occurred during evaluation
//
// The function follows this logic:
//  1. If AutoApprove is false in the template, returns false immediately
//  2. If AutoApprove is true but no ApprovalCondition is specified, returns true (auto-approve all)
//  3. If an ApprovalCondition is specified, evaluates the CEL expression with the request object
//     and returns the boolean result
//
// The approval condition is a CEL expression that has access to the request object via the
// "request" variable. The expression must evaluate to a boolean value.
func IsApproved(brt *capsulev1beta2.BreakRequestTemplate, br *capsulev1beta2.BreakRequest) (approved bool, err error) {
	if !brt.Spec.AutoApprove {
		return false, nil
	}

	if brt.Spec.ApprovalCondition == "" {
		return true, nil
	}

	prg, err := PrepareCondition(brt)
	if err != nil {
		return false, err
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(br)
	if err != nil {
		return false, err
	}

	result, _, err := prg.Eval(map[string]any{
		"request": obj,
	})
	if err != nil {
		return false, err
	}

	// Convert the result to boolean
	var ok bool
	approved, ok = result.Value().(bool)
	if !ok {
		return false, fmt.Errorf(
			"expression did not evaluate to a boolean, got: %T",
			result.Value(),
		)
	}

	return approved, nil
}

func PrepareCondition(brt *capsulev1beta2.BreakRequestTemplate) (cel.Program, error) {
	env, err := cel.NewEnv(
		cel.Variable("request", cel.DynType),
	)
	if err != nil {
		return nil, err
	}

	ast, iss := env.Compile(brt.Spec.ApprovalCondition)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}

	return env.Program(ast)
}
