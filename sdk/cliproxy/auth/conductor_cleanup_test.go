package auth

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
)

type cleanupTestStore struct {
	mu        sync.Mutex
	items     map[string]*Auth
	deletedID []string
}

func (s *cleanupTestStore) List(_ context.Context) ([]*Auth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]*Auth, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item.Clone())
	}
	return out, nil
}

func (s *cleanupTestStore) Save(_ context.Context, auth *Auth) (string, error) {
	if auth == nil {
		return "", nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.items == nil {
		s.items = make(map[string]*Auth)
	}
	s.items[auth.ID] = auth.Clone()
	return auth.ID, nil
}

func (s *cleanupTestStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, id)
	s.deletedID = append(s.deletedID, id)
	return nil
}

func TestMarkResult_RemovesIrrecoverableAuth(t *testing.T) {
	store := &cleanupTestStore{items: make(map[string]*Auth)}
	manager := NewManager(store, nil, nil)

	auth := &Auth{
		ID:       "cleanup-irrecoverable-auth",
		Provider: "codex",
	}
	if _, err := manager.Register(context.Background(), auth); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	reg := registry.GetGlobalRegistry()
	reg.UnregisterClient(auth.ID)
	reg.RegisterClient(auth.ID, auth.Provider, []*registry.ModelInfo{{ID: "gpt-5"}})
	t.Cleanup(func() {
		reg.UnregisterClient(auth.ID)
	})

	manager.MarkResult(context.Background(), Result{
		AuthID:       auth.ID,
		Provider:     auth.Provider,
		Model:        "gpt-5",
		Success:      false,
		ShouldDelete: true,
		Error: &Error{
			HTTPStatus: http.StatusUnauthorized,
			Message:    "authentication token has been invalidated",
		},
	})

	if _, ok := manager.GetByID(auth.ID); ok {
		t.Fatalf("expected auth %q to be removed from manager", auth.ID)
	}
	if got := manager.List(); len(got) != 0 {
		t.Fatalf("expected manager auth list to be empty, got %d entries", len(got))
	}

	store.mu.Lock()
	deleted := append([]string(nil), store.deletedID...)
	_, exists := store.items[auth.ID]
	store.mu.Unlock()
	if exists {
		t.Fatalf("expected auth %q to be removed from store", auth.ID)
	}
	if len(deleted) != 1 || deleted[0] != auth.ID {
		t.Fatalf("expected store delete to be called once for %q, got %v", auth.ID, deleted)
	}

	if models := reg.GetModelsForClient(auth.ID); len(models) != 0 {
		t.Fatalf("expected registry entry for %q to be removed, got %d models", auth.ID, len(models))
	}
}
