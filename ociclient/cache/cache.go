// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/gardener/component-cli/ociclient/metrics"
)

type layeredCache struct {
	log logr.Logger
	mux sync.RWMutex

	baseFs    vfs.FileSystem
	overlayFs vfs.FileSystem

	basePath string
}

// NewCache creates a new cache with the given options.
// It uses by default a tmp fs
func NewCache(log logr.Logger, options ...Option) (Cache, error) {
	opts := &Options{}
	opts = opts.ApplyOptions(options)

	if err := initBasePath(opts); err != nil {
		return nil, err
	}

	base, err := projectionfs.New(osfs.New(), opts.BasePath)
	if err != nil {
		return nil, err
	}
	var overlay vfs.FileSystem
	if opts.InMemoryOverlay {
		overlay = memoryfs.New()
	}

	//initialize metrics
	metrics.CachedItems.WithLabelValues(opts.BasePath).Set(0)
	metrics.CacheDiskUsage.WithLabelValues(opts.BasePath).Set(0)
	metrics.CacheHitsDisk.WithLabelValues(opts.BasePath).Add(0)
	if opts.InMemoryOverlay {
		metrics.CacheMemoryUsage.WithLabelValues(opts.BasePath).Set(0)
		metrics.CacheHitsMemory.WithLabelValues(opts.BasePath).Add(0)
	}

	err = vfs.Walk(base, "/", func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			metrics.CachedItems.WithLabelValues(opts.BasePath).Inc()
			metrics.CacheDiskUsage.WithLabelValues(opts.BasePath).Add(float64(info.Size()))
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error during initialization of cache metrics: %w", err)
	}

	return &layeredCache{
		log:       log,
		mux:       sync.RWMutex{},
		baseFs:    base,
		overlayFs: overlay,
		basePath:  opts.BasePath,
	}, nil
}

func initBasePath(opts *Options) error {
	if len(opts.BasePath) == 0 {
		path, err := ioutil.TempDir(os.TempDir(), "ocicache")
		if err != nil {
			return err
		}
		opts.BasePath = path
	}
	info, err := os.Stat(opts.BasePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(opts.BasePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if !info.IsDir() {
		return errors.New("path has to be a directory")
	}
	return nil
}

func (lc *layeredCache) Get(desc ocispecv1.Descriptor) (io.ReadCloser, error) {
	_, file, err := lc.get(path(desc))
	if err != nil {
		return nil, err
	}
	metrics.CacheHitsDisk.WithLabelValues(lc.basePath).Inc()
	return file, nil
}

func (lc *layeredCache) Add(desc ocispecv1.Descriptor, reader io.ReadCloser) error {
	path := path(desc)
	lc.mux.Lock()
	defer lc.mux.Unlock()
	defer reader.Close()

	file, err := lc.baseFs.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)

	// in case everything worked well, update metrics
	if err == nil {
		metrics.CachedItems.WithLabelValues(lc.basePath).Inc()
		if fileInfo, metricsErr := file.Stat(); metricsErr == nil {
			metrics.CacheDiskUsage.WithLabelValues(lc.basePath).Add(float64(fileInfo.Size()))
		} else {
			lc.log.V(7).Error(metricsErr, "Failed to access %q", file.Name())
		}
	}
	return err
}

//
//func (lc *layeredCache) info(dgst string) (os.FileInfo, error) {
//	lc.mux.RLock()
//	defer lc.mux.RUnlock()
//
//	// first search in the overlayFs layer
//	if lc.overlayFs != nil {
//		if info, err := lc.overlayFs.Stat(dgst); err == nil {
//			return info, nil
//		}
//	}
//
//	info, err := lc.baseFs.Stat(dgst)
//	if err != nil {
//		if os.IsNotExist(err) {
//			return nil, ErrNotFound
//		}
//		return nil, err
//	}
//	return info, nil
//}

func (lc *layeredCache) get(dgst string) (os.FileInfo, vfs.File, error) {
	lc.mux.RLock()
	defer lc.mux.RUnlock()

	// first search in the overlayFs layer
	if lc.overlayFs != nil {
		if info, err := lc.overlayFs.Stat(dgst); err == nil {
			file, err := lc.overlayFs.OpenFile(dgst, os.O_RDONLY, os.ModePerm)
			if err != nil {
				return nil, nil, err
			}
			metrics.CacheHitsMemory.WithLabelValues(lc.basePath).Inc()
			return info, file, err
		}
		lc.log.V(7).Info("not found in overlay cache", "dgst", dgst, "digest", dgst)
	}

	info, err := lc.baseFs.Stat(dgst)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	file, err := lc.baseFs.OpenFile(dgst, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, nil, err
	}

	// copy file to in memory cache
	if lc.overlayFs != nil {
		overlayFile, err := lc.overlayFs.Create(dgst)
		if err != nil {
			// do not return an error here as we are only unable to write to better cache
			lc.log.V(5).Info(err.Error())
			return info, file, nil
		}
		defer overlayFile.Close()
		if _, err := io.Copy(overlayFile, file); err != nil {
			// do not return an error here as we are only unable to write to better cache
			lc.log.V(5).Info(err.Error())
			return info, file, nil
		}
		if fileInfo, err := lc.overlayFs.Stat(dgst); err == nil {
			metrics.CacheMemoryUsage.WithLabelValues(lc.basePath).Add(float64(fileInfo.Size()))
		} else {
			lc.log.V(7).Error(err, "Failed to access %q", dgst)
		}
	}
	return info, file, nil
}

func path(desc ocispecv1.Descriptor) string {
	return desc.Digest.Encoded()
}
