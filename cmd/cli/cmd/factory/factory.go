// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Factory provides access to Kubernetes clients and configuration flags.
type Factory interface {
	ToRESTConfig() (*rest.Config, error)
	ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error)
	ToRESTMapper() (meta.RESTMapper, error)
	ToKubeClient() (kubernetes.Interface, error)
	ToControllerRuntimeClient() (client.Client, error)
	ToDynamicClient() (dynamic.Interface, error)
	Namespace() (string, bool, error)
	ConfigFlags() *genericclioptions.ConfigFlags
}
