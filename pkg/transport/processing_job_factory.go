// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package transport

import (
	"fmt"
	"time"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/go-logr/logr"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/pkg/transport/config"
	"github.com/gardener/component-cli/pkg/transport/process/downloaders"
	"github.com/gardener/component-cli/pkg/transport/process/processors"
	"github.com/gardener/component-cli/pkg/transport/process/uploaders"
)

// NewProcessingJobFactory creates a new processing job factory
func NewProcessingJobFactory(transportCfg config.ParsedTransportConfig, ociClient ociclient.Client, ocicache cache.Cache, targetCtx cdv2.OCIRegistryRepository, log logr.Logger, processorTimeout time.Duration) (*ProcessingJobFactory, error) {
	df := downloaders.NewDownloaderFactory(ociClient, ocicache)
	pf := processors.NewProcessorFactory(ocicache)
	uf := uploaders.NewUploaderFactory(ociClient, ocicache, targetCtx)

	f := ProcessingJobFactory{
		parsedConfig:      &transportCfg,
		downloaderFactory: df,
		processorFactory:  pf,
		uploaderFactory:   uf,
		log:               log,
		processorTimeout:  processorTimeout,
	}

	return &f, nil
}

// ProcessingJobFactory defines a helper struct for creating processing jobs
type ProcessingJobFactory struct {
	parsedConfig      *config.ParsedTransportConfig
	uploaderFactory   *uploaders.UploaderFactory
	downloaderFactory *downloaders.DownloaderFactory
	processorFactory  *processors.ProcessorFactory
	log               logr.Logger
	processorTimeout  time.Duration
}

// Create creates a new processing job for a resource
func (c *ProcessingJobFactory) Create(cd cdv2.ComponentDescriptor, res cdv2.Resource) (*ProcessingJob, error) {
	downloaderDefs := c.parsedConfig.MatchDownloaders(cd, res)
	downloaders := []NamedResourceStreamProcessor{}
	for _, dd := range downloaderDefs {
		p, err := c.downloaderFactory.Create(dd.Type, dd.Spec)
		if err != nil {
			return nil, fmt.Errorf("unable to create downloader: %w", err)
		}
		downloaders = append(downloaders, NamedResourceStreamProcessor{
			Name:      dd.Name,
			Processor: p,
		})
	}

	processingRuleDefs := c.parsedConfig.MatchProcessingRules(cd, res)
	processors := []NamedResourceStreamProcessor{}
	for _, rd := range processingRuleDefs {
		for _, pd := range rd.Processors {
			p, err := c.processorFactory.Create(pd.Type, pd.Spec)
			if err != nil {
				return nil, fmt.Errorf("unable to create processor: %w", err)
			}
			processors = append(processors, NamedResourceStreamProcessor{
				Name:      pd.Name,
				Processor: p,
			})
		}
	}

	uploaderDefs := c.parsedConfig.MatchUploaders(cd, res)
	uploaders := []NamedResourceStreamProcessor{}
	for _, ud := range uploaderDefs {
		p, err := c.uploaderFactory.Create(ud.Type, ud.Spec)
		if err != nil {
			return nil, fmt.Errorf("unable to create uploader: %w", err)
		}
		uploaders = append(uploaders, NamedResourceStreamProcessor{
			Name:      ud.Name,
			Processor: p,
		})
	}

	jobLog := c.log.WithValues("component-name", cd.Name, "component-version", cd.Version, "resource-name", res.Name, "resource-version", res.Version)
	job, err := NewProcessingJob(cd, res, downloaders, processors, uploaders, jobLog, c.processorTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to create processing job: %w", err)
	}

	return job, nil
}
