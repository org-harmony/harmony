package persistence

import "context"

// KVStore is a key-value store.
type KVStore[K any, V any] interface {
	Read(ctx context.Context, key K) (V, error)      // Read reads a value by key.
	Write(ctx context.Context, key K, value V) error // Write writes a value by key.
	Delete(ctx context.Context, key K) error         // Delete removes a value by key.
}
