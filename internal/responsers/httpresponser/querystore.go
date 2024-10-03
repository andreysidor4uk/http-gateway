package httpresponser

import (
	"sync"

	"github.com/google/uuid"
)

type queryStore struct {
	mu    sync.Mutex
	store map[uuid.UUID]*query
}

func (qs *queryStore) push(q *query) uuid.UUID {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	id := uuid.New()
	qs.store[id] = q
	go func() {
		if _, ok := <-q.req.Context().Done(); !ok {
			qs.pop(id)
		}
	}()
	return id
}

func (qs *queryStore) pop(id uuid.UUID) *query {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if _, ok := qs.store[id]; !ok {
		return nil
	}
	defer delete(qs.store, id)

	return qs.store[id]
}

func newQueryStore() *queryStore {
	return &queryStore{
		store: make(map[uuid.UUID]*query),
	}
}
