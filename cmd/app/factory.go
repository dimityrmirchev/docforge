// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gardener/docforge/pkg/api"
	"github.com/gardener/docforge/pkg/hugo"
	"github.com/gardener/docforge/pkg/metrics"
	"github.com/gardener/docforge/pkg/resourcehandlers"
	"github.com/gardener/docforge/pkg/writers"
	"github.com/hashicorp/go-multierror"

	//"github.com/gardener/docforge/pkg/metrics"
	"github.com/gardener/docforge/pkg/processors"
	"github.com/gardener/docforge/pkg/reactor"
	"github.com/gardener/docforge/pkg/resourcehandlers/fs"
	ghrs "github.com/gardener/docforge/pkg/resourcehandlers/github"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

// Options is the set of parameters for creating
// reactor objects
type Options struct {
	MaxWorkersCount              int
	MinWorkersCount              int
	FailFast                     bool
	DestinationPath              string
	ResourcesPath                string
	ResourceDownloadWorkersCount int
	RewriteEmbedded              bool
	GitHubTokens                 map[string]string
	Metering                     *Metering
	GitHubInfoPath               string
	DryRunWriter                 io.Writer
	Resolve                      bool
	ResourceHandlers             []resourcehandlers.URIValidator
	Hugo                         *hugo.Options
}

// Metering encapsulates options for setting up client-side
// metering
type Metering struct {
	Enabled bool
}

// NewReactor creates a Reactor from Options
func NewReactor(ctx context.Context, options *Options, globalLinksCfg *api.Links) (*reactor.Reactor, error) {
	dryRunWriters := writers.NewDryRunWritersFactory(options.DryRunWriter)
	o := &reactor.Options{
		MaxWorkersCount:              options.MaxWorkersCount,
		MinWorkersCount:              options.MinWorkersCount,
		FailFast:                     options.FailFast,
		DestinationPath:              options.DestinationPath,
		ResourcesPath:                options.ResourcesPath,
		ResourceDownloadWorkersCount: options.ResourceDownloadWorkersCount,
		RewriteEmbedded:              options.RewriteEmbedded,
		Processor:                    nil,
		ResourceHandlers:             options.ResourceHandlers,
		DryRunWriter:                 dryRunWriters,
		Resolve:                      options.Resolve,
		GlobalLinksConfig:            globalLinksCfg,
	}
	if options.DryRunWriter != nil {
		o.Writer = dryRunWriters.GetWriter(options.DestinationPath)
		o.ResourceDownloadWriter = dryRunWriters.GetWriter(filepath.Join(options.DestinationPath, options.ResourcesPath))
	} else {
		o.Writer = &writers.FSWriter{
			Root: options.DestinationPath,
		}
		o.ResourceDownloadWriter = &writers.FSWriter{
			Root: filepath.Join(options.DestinationPath, options.ResourcesPath),
		}
	}

	if len(options.GitHubInfoPath) > 0 {
		o.GitInfoWriter = &writers.FSWriter{
			Root: filepath.Join(options.DestinationPath, options.GitHubInfoPath),
			Ext:  "json",
		}
	}

	if options.Hugo != nil {
		WithHugo(o, options)
	}

	return reactor.NewReactor(o), nil
}

// WithHugo adapts the reactor.Options object with Hugo-specific
// settings for writer and processor
func WithHugo(reactorOptions *reactor.Options, o *Options) {
	hugoOptions := o.Hugo
	reactorOptions.Processor = &processors.ProcessorChain{
		Processors: []processors.Processor{
			&processors.FrontMatter{},
			hugo.NewProcessor(hugoOptions),
		},
	}
	if o.DryRunWriter != nil {
		hugoOptions.Writer = reactorOptions.Writer
	} else {
		hugoOptions.Writer = &writers.FSWriter{
			Root: filepath.Join(o.DestinationPath),
		}
	}
	reactorOptions.Writer = hugo.NewWriter(hugoOptions)
}

// initResourceHandlers initializes the resource handler
// objects used by reactors
func initResourceHandlers(ctx context.Context, githubTokens map[string]string, metering *Metering) ([]resourcehandlers.URIValidator, error) {
	rhs := []resourcehandlers.URIValidator{
		fs.NewFSResourceHandler(),
	}
	var errs *multierror.Error
	if githubTokens != nil {
		for instance, token := range githubTokens {
			if !strings.HasPrefix(instance, "https://") && !strings.HasPrefix(instance, "http://") {
				instance = "https://" + instance
			}

			p, err := url.Parse(instance)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("couldn't parse url: %s", instance))
				continue
			}

			ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
			oauthClient := oauth2.NewClient(ctx, ts)
			if metering != nil && metering.Enabled {
				// Wrap client
				oauthClient = metrics.InstrumentClientRoundTripperDuration(oauthClient)
			}

			if p.Host == "github.com" {
				client := github.NewClient(oauthClient)
				gh := ghrs.NewResourceHandler(client, []string{"github.com", "raw.githubusercontent.com"})
				rhs = append(rhs, gh)
				continue
			}

			client, err := github.NewEnterpriseClient(instance, "", oauthClient)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("cannot create GitHub enterprise client for instance %s", instance))
				continue
			}
			defaultRawHost := "raw." + p.Host
			gh := ghrs.NewResourceHandler(client, []string{p.Host, defaultRawHost})
			rhs = append(rhs, gh)
		}
	}
	return rhs, errs.ErrorOrNil()
}
