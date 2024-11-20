package sturdycache

import (
	"context"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	redis_store "github.com/eko/gocache/store/redis/v4"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/rand"
)

type ReplicatedCache struct {
	masterCache   *cache.Cache[any]
	replicaCaches []*cache.Cache[any]
	readCaches    []*cache.Cache[any]
}

// NewReplicatedCache creates a new gocache store instance for redis master-replica setups.
// Requires one master server address and one or more replica addresses.
func NewReplicatedCache(
	masterAddr string,
	replicaAddrs []string,
	password string,
) (*ReplicatedCache, error) {
	masterClient := redis.NewClient(&redis.Options{
		Addr:         masterAddr,
		Password:     password,
		DB:           0,
		PoolSize:     200,
		MinIdleConns: 10,
	})

	masterStore := redis_store.NewRedis(masterClient)
	masterCache := cache.New[any](masterStore)

	replicaCaches := []*cache.Cache[any]{}

	for _, addr := range replicaAddrs {
		replicaClient := redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     password,
			DB:           0,
			PoolSize:     200,
			MinIdleConns: 10,
		})

		replicaStore := redis_store.NewRedis(replicaClient)
		replicaCaches = append(replicaCaches, cache.New[any](replicaStore))
	}

	cache := &ReplicatedCache{
		masterCache:   masterCache,
		replicaCaches: replicaCaches,
		readCaches:    append(replicaCaches, masterCache),
	}

	return cache, nil
}

// Set writes to master.
func (rc *ReplicatedCache) Set(ctx context.Context, key any, value any, options ...store.Option) error {
	return rc.masterCache.Set(ctx, key, value, options...)
}

// Get reads from master and replica caches.
func (rc *ReplicatedCache) Get(ctx context.Context, key any) (any, error) {
	return rc.readCaches[rand.Intn(len(rc.readCaches))].Get(ctx, key)
}

// Invalidate master cache entries for given options.
func (rc *ReplicatedCache) Invalidate(ctx context.Context, options ...store.InvalidateOption) error {
	return rc.masterCache.Invalidate(ctx, options...)
}

// Delete from master cache.
func (rc *ReplicatedCache) Delete(ctx context.Context, key any) error {
	return rc.masterCache.Delete(ctx, key)
}

// Clear master cache.
func (rc *ReplicatedCache) Clear(ctx context.Context) error {
	return rc.masterCache.Clear(ctx)
}

// GetType returns cache type name.
func (rc *ReplicatedCache) GetType() string {
	return "replicated_redis"
}
