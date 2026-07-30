package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrometry/server/internal/db"
	"github.com/andrometry/server/internal/models"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestRequireScope_valid(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := RequireScope("agent:context:read")(next)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ScopesKey, "agent:context:read,agent:briefing:read")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ok" {
		t.Errorf("body = %q, want ok", w.Body.String())
	}
}

func TestRequireScope_insufficient(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := RequireScope("admin")(next)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ScopesKey, "agent:context:read")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireScope_noScopes(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := RequireScope("admin")(next)

	req := httptest.NewRequest("GET", "/", nil)
	// No scopes in context
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireAnyScope_firstMatch(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAnyScope("device:ingest", "agent:interventions:write")(next)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ScopesKey, "device:ingest,device:read_config")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireAnyScope_secondMatch(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAnyScope("device:ingest", "agent:interventions:write")(next)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ScopesKey, "agent:interventions:write,agent:context:read")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireAnyScope_noMatch(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := RequireAnyScope("device:ingest", "agent:interventions:write")(next)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ScopesKey, "agent:context:read")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireAnyScope_noScopes(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	handler := RequireAnyScope("device:ingest", "agent:interventions:write")(next)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireAnyScope_adminWildcard(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireAnyScope("device:ingest", "admin")(next)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ScopesKey, "admin")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCORS_setsHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CORS(next)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS origin = %q, want *", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS methods header not set")
	}
}

func TestCORS_preflight(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for OPTIONS")
	})

	handler := CORS(next)

	req := httptest.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestAuth_missingHeader(t *testing.T) {
	database := setupTestDB(t)
	handler := Auth(database)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_invalidFormat(t *testing.T) {
	database := setupTestDB(t)
	handler := Auth(database)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	handler(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_invalidToken(t *testing.T) {
	database := setupTestDB(t)
	handler := Auth(database)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer non-existent-token")
	w := httptest.NewRecorder()
	handler(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_validToken(t *testing.T) {
	database := setupTestDB(t)

	// Create a test token
	hash := db.HashToken("test-token-123")
	err := database.CreateToken(&models.AuthToken{
		TokenID:   "tok-1",
		UserID:    "primary",
		TokenHash: hash,
		TokenType: "device",
		Scopes:    "device:ingest,agent:context:read",
	})
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	handler := Auth(database)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context values
		uid, ok := r.Context().Value(UserIDKey).(string)
		if !ok || uid != "primary" {
			t.Errorf("UserID = %v, want primary", uid)
		}
		scopes, ok := r.Context().Value(ScopesKey).(string)
		if !ok || scopes != "device:ingest,agent:context:read" {
			t.Errorf("Scopes = %v, want device:ingest,agent:context:read", scopes)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")
	w := httptest.NewRecorder()
	handler(next).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLogging_recordsStatus(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("created"))
	})

	handler := Logging(next)

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if w.Body.String() != "created" {
		t.Errorf("body = %q, want created", w.Body.String())
	}
}

func TestRequireScope_wildcard(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequireScope("admin")(next)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), ScopesKey, "*")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuth_expiredToken(t *testing.T) {
	database := setupTestDB(t)

	// Create an expired token
	hash := db.HashToken("expired-token")
	err := database.CreateToken(&models.AuthToken{
		TokenID:   "tok-expired",
		UserID:    "primary",
		TokenHash: hash,
		TokenType: "agent",
		Scopes:    "agent:context:read",
		ExpiresAt: stringPtr(time.Now().Add(-1 * time.Hour).Format(time.RFC3339)),
	})
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	handler := Auth(database)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for expired token")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()
	handler(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (expired token should be rejected)", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_revokedToken(t *testing.T) {
	database := setupTestDB(t)

	// Create a token and then revoke it
	hash := db.HashToken("revoked-token")
	err := database.CreateToken(&models.AuthToken{
		TokenID:   "tok-revoked",
		UserID:    "primary",
		TokenHash: hash,
		TokenType: "device",
		Scopes:    "device:ingest",
	})
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Revoke it
	_, err = database.DB.Exec("UPDATE auth_tokens SET revoked_at = datetime('now') WHERE token_id = ?", "tok-revoked")
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	handler := Auth(database)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for revoked token")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer revoked-token")
	w := httptest.NewRecorder()
	handler(next).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (revoked token should be rejected)", w.Code, http.StatusUnauthorized)
	}
}

func stringPtr(s string) *string {
	return &s
}
