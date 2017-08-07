package iiif

import (
	"context"
	"github.com/golang/groupcache"
	"net/http"
)

// ContextKey is the cache key to use.
type ContextKey string

// WithGroupCaches sets the various caches.
func WithGroupCaches(h http.Handler, groups map[string]*groupcache.Group) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		for k, v := range groups {
			ctx = context.WithValue(ctx, ContextKey(k), v)
		}
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}

// WithConfig sets the IIIF server configuration.
func WithConfig(h http.Handler, config *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKey("config"), config)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}
