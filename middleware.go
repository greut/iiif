package main

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

// WithRootDirectory sets the root directory in the context.
func WithRootDirectory(h http.Handler, root string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKey("root"), root)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}
