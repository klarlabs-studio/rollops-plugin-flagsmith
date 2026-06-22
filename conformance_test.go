package flagsmith

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.klarlabs.de/rollops/pkg/flagconformance"
	"go.klarlabs.de/rollops/pkg/plugin"
)

// fakeFlagsmith accepts the lookup + patch sequence ApplyFlag performs.
func fakeFlagsmith(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/featurestates"):
			_, _ = w.Write([]byte(`{"results":[{"id":2}]}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/features"):
			_, _ = w.Write([]byte(`{"results":[{"id":1,"name":"checkout"}]}`))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestConformance(t *testing.T) {
	flagconformance.Run(t, func() (plugin.FlagProvider, error) {
		srv := fakeFlagsmith(t)
		return Provider{BaseURL: srv.URL, Token: "tok", ProjectID: "1", HTTP: srv.Client()}, nil
	}, plugin.FlagChange{Flag: "checkout", Environment: "production"})
}
