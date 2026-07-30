package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/andrometry/server/internal/db"
)

type contextKey string

const (
	TokenKey contextKey = "token"
	UserIDKey contextKey = "user_id"
	ScopesKey contextKey = "scopes"
	ActorTypeKey contextKey = "actor_type"
	ActorIDKey contextKey = "actor_id"
)

// Auth is middleware that validates bearer tokens and injects user/scope info.
func Auth(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			token := parts[1]
			hash := db.HashToken(token)

			tokenInfo, err := database.ValidateToken(hash)
			if err != nil {
				http.Error(w, `{"error":"invalid or revoked token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), TokenKey, tokenInfo)
			ctx = context.WithValue(ctx, UserIDKey, tokenInfo.UserID)
			ctx = context.WithValue(ctx, ScopesKey, tokenInfo.Scopes)
			ctx = context.WithValue(ctx, ActorTypeKey, tokenInfo.TokenType)
			ctx = context.WithValue(ctx, ActorIDKey, tokenInfo.TokenID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireScope checks that the request's token has the required scope.
func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes, ok := r.Context().Value(ScopesKey).(string)
			if !ok {
				http.Error(w, `{"error":"no scopes found"}`, http.StatusForbidden)
				return
			}

			if !db.HasScope(scopes, scope) {
				http.Error(w, `{"error":"insufficient scope: "+`+scope+`}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyScope checks that the request's token has at least one of the required scopes.
func RequireAnyScope(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenScopes, ok := r.Context().Value(ScopesKey).(string)
			if !ok {
				http.Error(w, `{"error":"no scopes found"}`, http.StatusForbidden)
				return
			}

			for _, required := range scopes {
				if db.HasScope(tokenScopes, required) {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"error":"insufficient scope: requires one of `+strings.Join(scopes, ", ")+`"}`, http.StatusForbidden)
		})
	}
}

// CORS adds permissive CORS headers for development.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, If-None-Match")
		w.Header().Set("Access-Control-Expose-Headers", "ETag")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging logs incoming requests with method, path, status code, and latency.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		log.Printf("[http] %s %s %d %s", r.Method, r.URL.Path, ww.status, time.Since(start).Round(time.Millisecond))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
