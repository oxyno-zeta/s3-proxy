package cache

import (
	"bytes"
	"context"
	"encoding/gob"
	"strings"

	"github.com/allegro/bigcache"
	"github.com/coocood/freecache"
	"github.com/eko/gocache/v2/cache"
	"github.com/eko/gocache/v2/store"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	gocache "github.com/patrickmn/go-cache"
)

type cac struct {
	cfgManager config.Manager
	ccache     *cache.Cache
}

func (c *cac) Initialize() error {
	// Get configuration
	cfg := c.cfgManager.GetConfig()

	// Initialize store
	var sto store.StoreInterface

	// Switch for supported caches
	switch cfg.Cache.Type {
	// Bigcache case
	case config.DefaultCacheBigcacheType:
		// Init bigcache
		bcache, err := bigcache.NewBigCache(bigcache.DefaultConfig(cfg.Cache.DefaultExpiration))
		// Check error
		if err != nil {
			return err
		}

		// Create BigCache store
		sto = store.NewBigcache(bcache, nil)
	// Go-cache case
	case config.DefaultCacheGocacheType:
		// Initialize go-cache
		gca := gocache.New(cfg.Cache.DefaultExpiration, cfg.Cache.GoCache.PurgeExpiration)

		// Create Store
		sto = store.NewGoCache(gca, nil)
	// Freecache case
	case config.DefaultCacheFreecacheType:
		// Initialize free cache
		fca := freecache.NewCache(cfg.Cache.Freecache.Size)

		// Create store
		sto = store.NewFreecache(fca, &store.Options{
			Expiration: cfg.Cache.DefaultExpiration,
		})
	}

	// Create cache from store
	cc := cache.New(sto)

	// Store cache
	c.ccache = cc

	// Default result
	return nil
}

func (c *cac) Reload() error {
	// Get configuration
	cfg := c.cfgManager.GetConfig()

	// Check if cache isn't declared
	if cfg.Cache == nil {
		// Check if previous cache is existing
		if c.ccache != nil {
			// Clear cache
			err := c.ccache.Clear(context.Background())
			// Check error
			if err != nil {
				return err
			}

			// Clean
			c.ccache = nil
		}

		// Stop here
		return nil
	}

	// Default result
	return c.Initialize()
}

func (c *cac) Get(ctx context.Context, key string, result interface{}) error {
	// Check if cache exists
	if c.ccache == nil {
		// Nothing
		return nil
	}

	// Get value from cache
	res, err := c.ccache.Get(ctx, key)
	// Check error
	if err != nil {
		// Check if error if a not found error
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "nable to retrieve") {
			// Not an error
			return nil
		}

		// Return error
		return err
	}

	// Create buffer from cache result
	buf := bytes.NewBuffer([]byte(res.(string)))
	// Create decoder from buffer
	dec := gob.NewDecoder(buf)

	// Decode in result
	err = dec.Decode(&result)
	// Check error
	if err != nil {
		return err
	}

	// Default case
	return nil
}

func (c *cac) Set(ctx context.Context, key string, value interface{}) error {
	// Check if cache exists
	if c.ccache == nil {
		// Nothing
		return nil
	}

	// Create buffer
	var buf bytes.Buffer
	// Create gob encoder with buffer
	enc := gob.NewEncoder(&buf)

	// Encode value
	err := enc.Encode(value)
	// Check error
	if err != nil {
		return err
	}

	// Store in cache
	err = c.ccache.Set(ctx, key, buf.Bytes(), nil)
	// Check error
	if err != nil {
		return err
	}

	// Default case
	return nil
}
