package cache

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/stretchr/testify/require"
)

func TestCacheItemReplacesDueToCostOverflow(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	configMap1 := entity.ConfigMap{
		Metadata: entity.Metadata{Name: "test-1", Namespace: "test-namespace"},
		Data:     map[string]string{"key-1": "value-1"},
	}
	configMap2 := entity.ConfigMap{
		Metadata: entity.Metadata{Name: "test-2", Namespace: "test-namespace"},
		Data:     map[string]string{"key-2": "value-2"},
	}
	storeMaxSize := (int64)(250)
	cache, err := NewResourcesCache(2, storeMaxSize, 0, 0)
	assertions.NoError(err)

	ok, err := cache.ConfigMaps.Set(ctx, configMap1)
	assertions.NoError(err)
	assertions.True(ok)

	configMapFromCache1 := cache.ConfigMaps.Get(ctx, configMap1.Namespace, configMap1.Name)
	assertions.NotNil(configMap1)
	assertions.Equal(configMap1.Data, configMapFromCache1.Data)

	ok, err = cache.ConfigMaps.Set(ctx, configMap2)
	assertions.NoError(err)
	assertions.True(ok)

	configMapFromCache1 = cache.ConfigMaps.Get(ctx, configMap1.Namespace, configMap1.Name)
	assertions.Nil(configMapFromCache1)

	configMapFromCache2 := cache.ConfigMaps.Get(ctx, configMap2.Namespace, configMap2.Name)
	assertions.NotNil(configMapFromCache2)
	assertions.Equal(configMap2.Data, configMapFromCache2.Data)
}

func TestCacheItemIgnoredDueToMaxItemSize(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	configMap1 := entity.ConfigMap{
		Metadata: entity.Metadata{Name: "test-1", Namespace: "test-namespace"},
		Data:     map[string]string{"key-1": "value-1"},
	}
	storeMaxSize := (int64)(250)
	maxItemSizeInBytes := 50

	bytes1, err := json.Marshal(configMap1)
	assertions.NoError(err)

	assertions.True(len(bytes1) > maxItemSizeInBytes)

	cache, err := NewResourcesCache(2, storeMaxSize, int64(maxItemSizeInBytes), 0)
	assertions.NoError(err)

	ok, err := cache.ConfigMaps.Set(ctx, configMap1)
	assertions.Error(err)
	assertions.False(ok)
	assertions.Contains(err.Error(), "failed to place")

	configMapFromCache1 := cache.ConfigMaps.Get(ctx, configMap1.Namespace, configMap1.Name)
	assertions.Nil(configMapFromCache1)
}

func TestCacheItemCRUD(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	name := "test-1"
	namespace := "test-namespace"
	configMap1 := entity.ConfigMap{
		Metadata: entity.Metadata{Name: name, Namespace: namespace},
		Data:     map[string]string{"key-1": "value-1"},
	}
	configMap2 := configMap1
	configMap2.Data["key-1"] = "value-2"

	storeMaxSize := (int64)(250)
	cache, err := NewResourcesCache(2, storeMaxSize, 250, 0)
	assertions.NoError(err)

	ok, err := cache.ConfigMaps.Set(ctx, configMap1)
	assertions.NoError(err)
	assertions.True(ok)

	configMapFromCache1 := cache.ConfigMaps.Get(ctx, namespace, name)
	assertions.NotNil(configMapFromCache1)
	assertions.Equal(configMap1.Data, configMapFromCache1.Data)

	ok, err = cache.ConfigMaps.Set(ctx, configMap2)
	assertions.NoError(err)
	assertions.True(ok)

	configMapFromCache2 := cache.ConfigMaps.Get(ctx, namespace, name)
	assertions.NotNil(configMapFromCache2)
	assertions.Equal(configMap2.Data, configMapFromCache2.Data)

	cache.ConfigMaps.Delete(ctx, namespace, name)

	configMapFromCache3 := cache.ConfigMaps.Get(ctx, namespace, name)
	assertions.Nil(configMapFromCache3)
}

func TestCacheItemsFromDifferentNamespaces(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	name := "test-1"
	namespace1 := "test-namespace-1"
	namespace2 := "test-namespace-2"
	configMap1 := entity.ConfigMap{
		Metadata: entity.Metadata{Name: name, Namespace: namespace1},
		Data:     map[string]string{"key-1": "value-1"},
	}
	configMap2 := entity.ConfigMap{
		Metadata: entity.Metadata{Name: name, Namespace: namespace2},
		Data:     map[string]string{"key-2": "value-2"},
	}

	storeMaxSize := (int64)(2000)
	cache, err := NewResourcesCache(2, storeMaxSize, 1000, 0)
	assertions.NoError(err)

	ok, err := cache.ConfigMaps.Set(ctx, configMap1)
	assertions.NoError(err)
	assertions.True(ok)

	ok, err = cache.ConfigMaps.Set(ctx, configMap2)
	assertions.NoError(err)
	assertions.True(ok)

	configMapFromCache1 := cache.ConfigMaps.Get(ctx, namespace1, name)
	assertions.NotNil(configMapFromCache1)
	assertions.Equal(configMap1.Data, configMapFromCache1.Data)

	configMapFromCache2 := cache.ConfigMaps.Get(ctx, namespace2, name)
	assertions.NotNil(configMapFromCache2)
	assertions.Equal(configMap2.Data, configMapFromCache2.Data)
}

func TestCacheItemsFromDifferentKinds(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	name := "test-1"
	namespace := "test-namespace-1"
	configMap := entity.ConfigMap{
		Metadata: entity.Metadata{Name: name, Namespace: namespace},
		Data:     map[string]string{"key-1": "value-1"},
	}
	secret := entity.Secret{
		Metadata: entity.Metadata{Name: name, Namespace: namespace},
		Data:     map[string][]byte{"key-2": []byte("value-2")},
		Type:     "test-type",
	}

	storeMaxSize := (int64)(2000)
	cache, err := NewResourcesCache(2, storeMaxSize, 1000, 0)
	assertions.NoError(err)

	ok, err := cache.ConfigMaps.Set(ctx, configMap)
	assertions.NoError(err)
	assertions.True(ok)

	ok, err = cache.Secrets.Set(ctx, secret)
	assertions.NoError(err)
	assertions.True(ok)

	configMapFromCache := cache.ConfigMaps.Get(ctx, namespace, name)
	assertions.NotNil(configMapFromCache)
	assertions.Equal(configMap.Data, configMapFromCache.Data)

	secretFromCache := cache.Secrets.Get(ctx, namespace, name)
	assertions.NotNil(secretFromCache)
	assertions.Equal(secret.Data, secretFromCache.Data)
	assertions.Equal(secret.Type, secretFromCache.Type)
}
