// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// VerifyResponse is a helper function to verify the response of a webhook function.
func VerifyResponse(response *admission.Response, status int32, message string) {
	ginkgo.GinkgoT().Helper()
	gomega.Expect(response.Result).NotTo(gomega.BeNil())
	gomega.Expect(response.Result.Code).To(gomega.Equal(status))
	gomega.Expect(response.Result.Message).To(gomega.ContainSubstring(message))
}
