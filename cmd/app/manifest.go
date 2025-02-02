// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gardener/docforge/pkg/api"
	"github.com/gardener/docforge/pkg/resourcehandlers"
)

// Manifest reads the resource at uri, resolves it as template applying vars,
// and finally parses it into api.Documentation model
func manifest(ctx context.Context, uri string, resourceHandlers []resourcehandlers.ResourceHandler, flagsVars map[string]string, hugoEnabled bool) (*api.Documentation, error) {
	var (
		handler         resourcehandlers.ResourceHandler
		manifestContent []byte
	)
	uri = strings.TrimSpace(uri)
	registry := resourcehandlers.NewRegistry(resourceHandlers...)

	//check if uri is in file system
	fileInfo, err := os.Stat(uri)
	if err == nil {
		//uri is from file system

		if fileInfo.IsDir() {
			return nil, fmt.Errorf("top level manifest %s is a directory", uri)
		}

		if manifestContent, err = ioutil.ReadFile(uri); err != nil {
			return nil, err
		}

		doc, err := api.ParseWithMetadata(manifestContent, "master", flagsVars, hugoEnabled)
		if err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %s. %+v", uri, err)
		}

		return doc, nil
	}

	if handler = registry.Get(uri); handler == nil {
		return nil, fmt.Errorf("no suitable reader found for %s. Is this path correct?", uri)
	}
	return handler.ResolveDocumentation(ctx, uri)
}
