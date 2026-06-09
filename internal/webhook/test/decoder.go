// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Decoder[T interface {
	runtime.Object
	DeepCopyInto(T)
}] struct {
	Object    T
	OldObject T
}

func copyInto[T interface {
	runtime.Object
	DeepCopyInto(T)
}](source T, into runtime.Object) {
	ginkgo.GinkgoT().Helper()

	if into == nil || runtime.Object(source) == nil {
		return
	}

	target, ok := into.(T)
	gomega.Expect(ok).To(gomega.BeTrue(), "type assertion failed: into is not of type T")
	source.DeepCopyInto(target)
}

func (d *Decoder[T]) Decode(_ admission.Request, into runtime.Object) error {
	ginkgo.GinkgoT().Helper()
	copyInto[T](d.Object, into)

	return nil
}

func (d *Decoder[T]) DecodeRaw(_ runtime.RawExtension, into runtime.Object) error {
	ginkgo.GinkgoT().Helper()
	copyInto[T](d.OldObject, into)

	return nil
}
