package knowledgebase

import (
	"fmt"

	"github.com/liliang-cn/cortexdb/v2/pkg/cortexdb"
)

// Compile-time check that our adapter satisfies CortexDB's embedder interface.
var _ cortexdb.Embedder = (*ProviderEmbedder)(nil)

// Store wraps a CortexDB instance.
type Store struct {
	db *cortexdb.DB
}

// Open opens (or creates) the CortexDB file at path, using embedder for vectorization.
func Open(path string, embedder cortexdb.Embedder) (*Store, error) {
	cfg := cortexdb.DefaultConfig(path)
	db, err := cortexdb.Open(cfg, cortexdb.WithEmbedder(embedder))
	if err != nil {
		return nil, fmt.Errorf("open cortexdb: %w", err)
	}
	return &Store{db: db}, nil
}

// DB exposes the underlying CortexDB handle.
func (s *Store) DB() *cortexdb.DB { return s.db }

// Healthy reports whether the store is open.
func (s *Store) Healthy() bool { return s != nil && s.db != nil }

// Close closes the store.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
