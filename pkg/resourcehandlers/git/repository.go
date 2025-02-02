// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"context"
	"fmt"
	"sync"

	"github.com/gardener/docforge/pkg/resourcehandlers"
	"github.com/go-git/go-git/v5/plumbing/transport"

	"github.com/gardener/docforge/pkg/resourcehandlers/git/gitinterface"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// State defines state of a git repo
type State int

const (
	_ State = iota
	// Prepared repo state
	Prepared
	// Failed repo state
	Failed
)

// Repository represents a git repo
type Repository struct {
	Auth          http.AuthMethod
	LocalPath     string
	RemoteURL     string
	State         State
	PreviousError error
	Git           gitinterface.Git

	mutex sync.RWMutex
}

// Prepare prepares the git repo for usage
func (r *Repository) Prepare(ctx context.Context, version string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	switch r.State {
	case Failed:
		return r.PreviousError
	case Prepared:
		return nil
	}

	if err := r.prepare(ctx, version); err != nil {
		r.State = Failed
		r.PreviousError = err
		return err
	}
	r.State = Prepared
	return nil
}

func (r *Repository) prepare(ctx context.Context, version string) error {
	repository, fetch, err := r.repository(ctx)
	if err != nil {
		return err
	}

	if fetch {
		if err := repository.FetchContext(ctx, &gogit.FetchOptions{
			Auth:       r.Auth,
			RemoteName: gogit.DefaultRemoteName,
		}); err != nil && err != gogit.NoErrAlreadyUpToDate {
			if err == transport.ErrRepositoryNotFound {
				return resourcehandlers.ErrResourceNotFound(r.RemoteURL)
			}
			return fmt.Errorf("failed to fetch repository %s: %v", r.LocalPath, err)
		}
	}

	w, err := repository.Worktree()
	if err != nil {
		return err
	}

	if err := w.Checkout(&gogit.CheckoutOptions{
		Branch: getCheckoutReferenceName(repository, version),
		Force:  true,
	}); err != nil {
		return fmt.Errorf("couldn't checkout version %s for repository %s: %v", version, r.LocalPath, err)
	}
	return nil
}

func (r *Repository) repository(ctx context.Context) (gitinterface.Repository, bool, error) {
	gitRepo, err := r.Git.PlainOpen(r.LocalPath)
	if err != nil {
		if err != gogit.ErrRepositoryNotExists {
			return nil, false, err
		}
		if gitRepo, err = r.Git.PlainCloneContext(ctx, r.LocalPath, false, &gogit.CloneOptions{
			URL:        r.RemoteURL,
			RemoteName: gogit.DefaultRemoteName,
			Auth:       r.Auth,
		}); err != nil {
			if err == transport.ErrRepositoryNotFound {
				return nil, false, resourcehandlers.ErrResourceNotFound(r.RemoteURL)
			}
			return nil, false, fmt.Errorf("failed to prepare repo: %s, %v", r.LocalPath, err)
		}
		return gitRepo, false, nil
	}
	return gitRepo, true, nil
}

func getCheckoutReferenceName(repository gitinterface.Repository, version string) plumbing.ReferenceName {
	var checkoutDestination plumbing.ReferenceName
	branchReference := plumbing.NewRemoteReferenceName(gogit.DefaultRemoteName, version)
	tagReference := plumbing.NewTagReferenceName(version)
	_, err1 := repository.Reference(branchReference, true)
	_, err2 := repository.Reference(tagReference, true)
	if err1 == nil {
		checkoutDestination = branchReference
	} else if err2 == nil {
		checkoutDestination = tagReference
	}
	return checkoutDestination
}
