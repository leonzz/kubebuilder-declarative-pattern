/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package loaders

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	addonsv1alpha1 "sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/apis/v1alpha1"
)

var FlagChannel = "./channels"

func init() {
	// TODO: Yuk - global flags are ugly
	flag.StringVar(&FlagChannel, "channel", FlagChannel, "location of channel to use")
}

type ManifestLoader struct {
	repo Repository
}

// NewManifestLoader provides a Repository that resolves versions based on an Addon object
// and loads manifests from the filesystem.
func NewManifestLoader(channel string) (*ManifestLoader, error) {
	if strings.HasPrefix(channel, "http://") || strings.HasPrefix(channel, "https://") {
		repo := NewHTTPRepository(channel)
		return &ManifestLoader{repo: repo}, nil
	}

	repo := NewFSRepository(channel)
	return &ManifestLoader{repo: repo}, nil
}

func (c *ManifestLoader) ResolveManifest(ctx context.Context, object runtime.Object) (map[string]string, error) {
	log := log.Log

	addonObject, ok := object.(addonsv1alpha1.CommonObject)
	if !ok {
		return nil, fmt.Errorf("object %T was not an addonsv1alpha1.CommonObject", object)
	}

	componentName := addonObject.ComponentName()

	spec := addonObject.CommonSpec()
	version := spec.Version
	channelName := spec.Channel

	// TODO: We should actually do id (1.1.2-aws or 1.1.1-nginx). But maybe YAGNI
	id := version

	if id == "" {
		// TODO: Put channel in spec
		if channelName == "" {
			channelName = "stable"
		}

		channel, err := c.repo.LoadChannel(ctx, channelName)
		if err != nil {
			return nil, err
		}

		version, err := channel.Latest(componentName)
		if err != nil {
			return nil, err
		}

		// TODO: We should probably copy the kubelet componentconfig

		if version == nil {
			return nil, fmt.Errorf("could not find latest version in channel %q", channelName)
		}
		id = version.Version

		log.WithValues("channel", channelName).WithValues("version", id).Info("resolved version from channel")
	} else {
		log.WithValues("version", version).Info("using specified version")
	}
	s := make(map[string]string)
	s, err := c.repo.LoadManifest(ctx, componentName, id)
	if err != nil {
		return nil, fmt.Errorf("error loading manifest: %v", err)
	}

	return s, nil
}
