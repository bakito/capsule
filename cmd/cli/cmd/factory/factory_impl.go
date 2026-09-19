// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
)

type factoryImpl struct {
	configFlags *genericclioptions.ConfigFlags
	scheme      *runtime.Scheme
}

// NewFactory returns a new Factory implementation backed by ConfigFlags.
func NewFactory(configFlags *genericclioptions.ConfigFlags) Factory {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(capsulev1beta2.AddToScheme(scheme))

	return &factoryImpl{
		configFlags: configFlags,
		scheme:      scheme,
	}
}

func (f *factoryImpl) ConfigFlags() *genericclioptions.ConfigFlags {
	return f.configFlags
}

func (f *factoryImpl) ToRESTConfig() (*rest.Config, error) {
	return f.configFlags.ToRESTConfig()
}

func (f *factoryImpl) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return f.configFlags.ToDiscoveryClient()
}

func (f *factoryImpl) ToRESTMapper() (meta.RESTMapper, error) {
	return f.configFlags.ToRESTMapper()
}

func (f *factoryImpl) ToKubeClient() (kubernetes.Interface, error) {
	cfg, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

func (f *factoryImpl) ToDynamicClient() (dynamic.Interface, error) {
	cfg, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	return dynamic.NewForConfig(cfg)
}

func (f *factoryImpl) ToControllerRuntimeClient() (client.Client, error) {
	cfg, err := f.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	return client.New(cfg, client.Options{
		Scheme: f.scheme,
	})
}

func (f *factoryImpl) Namespace() (string, bool, error) {
	return f.configFlags.ToRawKubeConfigLoader().Namespace()
}
