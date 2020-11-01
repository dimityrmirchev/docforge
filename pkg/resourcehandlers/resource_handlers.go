// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package resourcehandlers

import (
	"context"
	"errors"
	"reflect"

	"github.com/gardener/docforge/pkg/api"
)

// ErrIsResourceNotFound indicated that a resource was not found
var ErrResourceNotFound = errors.New("resource not found")

// ResourceHandler does resource specific operations on a type of objects
// identified by an uri schema that it accepts to handle
type ResourceHandler interface {
	URIValidator
	NodeResolver
	LinkControl
}

// URIValidator is implemented by Registry entries and
// used by its Get method to find an accepting entry to
// process a URI.
type URIValidator interface {
	// Accepts manifests if this ResourceHandler can manage the type of resources
	// identified by the URI scheme of uri.
	Accept(uri string) bool
}

// NodeResolver handles model elements such as nodeSelector that
// implicitly model node structures and need to be resolved in explicit ones.
type NodeResolver interface {
	// ResolveNodeSelector resolves the NodeSelector rules of a Node into subnodes
	// hierarchy (Node.Nodes)
	ResolveNodeSelector(ctx context.Context, node *api.Node, excludePaths []string, frontMatter map[string]interface{}, excludeFrontMatter map[string]interface{}, depth int32) ([]*api.Node, error)
	// ResolveDocumentation for a given uri
	ResolveDocumentation(ctx context.Context, uri string) (*api.Documentation, error)
}

// LinkControl implements specific operations on a URL
// required to reconcile links
type LinkControl interface {
	// ResourceName returns a breakdown of a resource name in the link, consisting
	// of name and potentially and extention without the dot.
	ResourceName(link string) (string, string)
	// BuildAbsLink should return an absolute path of a relative link in regards of the provided
	// source
	BuildAbsLink(source, link string) (string, error)
	// GetRawFormatLink returns a link to an embedable object (image) in raw format.
	// If the provided link is not referencing an embedable object, the function
	// returns absLink without changes.
	GetRawFormatLink(absLink string) (string, error)
	// SetVersion sets version to absLink according to the API scheme. For GitHub
	// for example this would replace e.g. the 'master' segment in the path with version
	SetVersion(absLink, version string) (string, error)
}

// Registry can register and return resource handlers
// for a url
type Registry interface {
	Load(rhs ...URIValidator)
	Get(uri string) URIValidator
	Remove(rh ...URIValidator)
	List() []URIValidator
}

type registry struct {
	handlers []URIValidator
}

// NewRegistry creates Registry object, optionally loading it with
// resourceHandlers if provided
func NewRegistry(resourceHandlers ...URIValidator) Registry {
	r := &registry{
		handlers: []URIValidator{},
	}
	if len(resourceHandlers) > 0 {
		r.Load(resourceHandlers...)
	}
	return r
}

// Load loads a ResourceHandler into the Registry
func (r *registry) Load(rhs ...URIValidator) {
	r.handlers = append(r.handlers, rhs...)
}

// Get returns an appropriate handler for this type of URIs if any
// one those registered accepts it (its Accepts method returns true).
func (r *registry) Get(uri string) URIValidator {
	for _, h := range r.handlers {
		if h.Accept(uri) {
			return h
		}
	}
	return nil
}

// List retruns the regsitered handlers in the order
// of their registration
func (r *registry) List() []URIValidator {
	return r.handlers
}

// Remove removes a ResourceHandler from registry. If no argument is provided
// the method will remove all registered handlers
func (r *registry) Remove(resourceHandlers ...URIValidator) {
	if len(resourceHandlers) == 0 {
		r.handlers = []URIValidator{}
	}
	idx := []int{}
	rhs := append([]URIValidator{}, resourceHandlers...)
	for _, rh := range rhs {
		if i := indexOf(rh, r.handlers); i > -1 {
			idx = append(idx, i)
		}
	}
	for _, i := range idx {
		remove(r.handlers, i)
	}
}

func indexOf(r URIValidator, rhs []URIValidator) int {
	var idx = -1
	for i, _r := range rhs {
		if reflect.DeepEqual(_r, r) {
			return i
		}
	}
	return idx
}

func remove(rhs []URIValidator, i int) []URIValidator {
	rhs[len(rhs)-1], rhs[i] = rhs[i], rhs[len(rhs)-1]
	return rhs[:len(rhs)-1]
}
