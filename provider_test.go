package flagsmith

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.klarlabs.de/rollops/pkg/plugin"
)

func TestApplyFlag_ResolvesAndPatches(t *testing.T) {
	var patched struct {
		path string
		body map[string]any
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case strings.Contains(r.URL.Path, "/projects/7/features/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{
				{"id": 42, "name": "other"}, {"id": 99, "name": "checkout"},
			}})
		case strings.Contains(r.URL.Path, "/environments/env-key/featurestates/") && r.Method == http.MethodGet:
			if r.URL.Query().Get("feature") != "99" {
				t.Errorf("feature filter = %q, want 99", r.URL.Query().Get("feature"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 555}}})
		case strings.Contains(r.URL.Path, "/environments/env-key/featurestates/555/") && r.Method == http.MethodPatch:
			patched.path = r.URL.Path
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &patched.body)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.Error(w, "nope", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := Provider{BaseURL: srv.URL, Token: "tok", ProjectID: "7", HTTP: srv.Client()}
	err := p.ApplyFlag(context.Background(), plugin.FlagChange{Flag: "checkout", Environment: "env-key", Percentage: 25})
	if err != nil {
		t.Fatalf("ApplyFlag: %v", err)
	}
	if patched.path != "/api/v1/environments/env-key/featurestates/555/" && !strings.HasSuffix(patched.path, "/featurestates/555/") {
		t.Errorf("patched wrong state: %s", patched.path)
	}
	if patched.body["enabled"] != true || patched.body["feature_state_value"] != "25" {
		t.Errorf("patch body = %+v, want enabled=true value=25", patched.body)
	}
}

func TestApplyFlag_DisabledClearsEnabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/features/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 1, "name": "f"}}})
		case strings.Contains(r.URL.Path, "/featurestates/") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": 2}}})
		case r.Method == http.MethodPatch:
			var body map[string]any
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			if body["enabled"] != false {
				t.Errorf("disabled change must set enabled=false, got %v", body["enabled"])
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()
	p := Provider{BaseURL: srv.URL, Token: "tok", ProjectID: "1", HTTP: srv.Client()}
	if err := p.ApplyFlag(context.Background(), plugin.FlagChange{Flag: "f", Environment: "e", Percentage: 0, Disabled: true}); err != nil {
		t.Fatal(err)
	}
}

func TestApplyFlag_RequiresCreds(t *testing.T) {
	if err := (Provider{}).ApplyFlag(context.Background(), plugin.FlagChange{Flag: "f"}); err == nil {
		t.Fatal("missing token/project must error")
	}
}

func TestApplyFlag_FeatureNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{}})
	}))
	defer srv.Close()
	p := Provider{BaseURL: srv.URL, Token: "t", ProjectID: "1", HTTP: srv.Client()}
	if err := p.ApplyFlag(context.Background(), plugin.FlagChange{Flag: "ghost", Environment: "e"}); err == nil {
		t.Fatal("unknown feature must error")
	}
}
