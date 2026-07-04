package mcpadapter

import (
	"context"
	"maps"
	"net/url"
	"slices"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SubscriptionRegistry tracks resource subscriptions by session and URI.
type SubscriptionRegistry struct {
	mu        sync.RWMutex
	bySession map[string]map[string]struct{}
	byURI     map[string]map[string]struct{}
}

// NewSubscriptionRegistry creates an empty resource subscription registry.
func NewSubscriptionRegistry() *SubscriptionRegistry {
	return &SubscriptionRegistry{
		bySession: map[string]map[string]struct{}{},
		byURI:     map[string]map[string]struct{}{},
	}
}

// Subscribe records a session subscription for a valid i18n resource URI.
func (r *SubscriptionRegistry) Subscribe(_ context.Context, req *mcp.SubscribeRequest) error {
	if err := validateI18nURI(req.Params.URI); err != nil {
		return err
	}

	sessionID := req.Session.ID()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bySession[sessionID] == nil {
		r.bySession[sessionID] = map[string]struct{}{}
	}
	if r.byURI[req.Params.URI] == nil {
		r.byURI[req.Params.URI] = map[string]struct{}{}
	}
	r.bySession[sessionID][req.Params.URI] = struct{}{}
	r.byURI[req.Params.URI][sessionID] = struct{}{}
	return nil
}

// Unsubscribe removes a session subscription for a valid i18n resource URI.
func (r *SubscriptionRegistry) Unsubscribe(_ context.Context, req *mcp.UnsubscribeRequest) error {
	if err := validateI18nURI(req.Params.URI); err != nil {
		return err
	}

	sessionID := req.Session.ID()
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.bySession[sessionID], req.Params.URI)
	delete(r.byURI[req.Params.URI], sessionID)
	if len(r.bySession[sessionID]) == 0 {
		delete(r.bySession, sessionID)
	}
	if len(r.byURI[req.Params.URI]) == 0 {
		delete(r.byURI, req.Params.URI)
	}
	return nil
}

// URIsForSession returns the resource URIs subscribed by sessionID.
func (r *SubscriptionRegistry) URIsForSession(sessionID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Collect(maps.Keys(r.bySession[sessionID]))
}

func validateI18nURI(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "i18n" {
		return mcp.ResourceNotFoundError(raw)
	}
	return nil
}
