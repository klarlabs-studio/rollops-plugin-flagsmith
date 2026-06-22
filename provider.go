// Package flagsmith is a Rollops feature-flag provider plugin backed by
// Flagsmith's Admin API. It drives a flag's enabled state and rollout
// percentage (carried as the flag's remote-config value) to match a rollout's
// progressive steps, so a Flagsmith flag tracks a Rollops canary in lockstep.
package flagsmith

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"go.klarlabs.de/rollops/pkg/plugin"
)

// Provider talks to Flagsmith's Admin API. BaseURL, Token, and ProjectID come
// from the plugin's environment (see Config); Environment is supplied per call
// by Rollops as the Flagsmith environment API key.
type Provider struct {
	BaseURL   string // e.g. https://api.flagsmith.com/api/v1
	Token     string // Admin API token (Authorization: Token <token>)
	ProjectID string // numeric Flagsmith project id, for feature lookup
	HTTP      *http.Client
}

func (p Provider) client() *http.Client {
	if p.HTTP != nil {
		return p.HTTP
	}
	return http.DefaultClient
}

// ApplyFlag sets the flag's enabled state (from !Disabled) and writes the
// rollout percentage as the flag's value in the given environment. It resolves
// the feature id by name, then the environment's feature-state id, then PATCHes
// it — the minimal-lookup automation flow Flagsmith documents.
func (p Provider) ApplyFlag(ctx context.Context, c plugin.FlagChange) error {
	if p.Token == "" || p.ProjectID == "" {
		return fmt.Errorf("flagsmith: FLAGSMITH_TOKEN and FLAGSMITH_PROJECT_ID are required")
	}
	featureID, err := p.featureID(ctx, c.Flag)
	if err != nil {
		return err
	}
	stateID, err := p.featureStateID(ctx, c.Environment, featureID)
	if err != nil {
		return err
	}
	return p.patchState(ctx, c.Environment, stateID, !c.Disabled, c.Percentage)
}

type results[T any] struct {
	Results []T `json:"results"`
}

type feature struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (p Provider) featureID(ctx context.Context, flag string) (int, error) {
	u := fmt.Sprintf("%s/projects/%s/features/?search=%s", p.BaseURL, p.ProjectID, url.QueryEscape(flag))
	var out results[feature]
	if err := p.do(ctx, http.MethodGet, u, nil, &out); err != nil {
		return 0, fmt.Errorf("flagsmith: lookup feature %q: %w", flag, err)
	}
	for _, f := range out.Results {
		if f.Name == flag {
			return f.ID, nil
		}
	}
	return 0, fmt.Errorf("flagsmith: feature %q not found in project %s", flag, p.ProjectID)
}

type featureState struct {
	ID int `json:"id"`
}

func (p Provider) featureStateID(ctx context.Context, envKey string, featureID int) (int, error) {
	u := fmt.Sprintf("%s/environments/%s/featurestates/?feature=%d", p.BaseURL, url.PathEscape(envKey), featureID)
	var out results[featureState]
	if err := p.do(ctx, http.MethodGet, u, nil, &out); err != nil {
		return 0, fmt.Errorf("flagsmith: lookup feature state: %w", err)
	}
	if len(out.Results) == 0 {
		return 0, fmt.Errorf("flagsmith: no feature state for feature %d in environment %q", featureID, envKey)
	}
	return out.Results[0].ID, nil
}

func (p Provider) patchState(ctx context.Context, envKey string, stateID int, enabled bool, percentage int) error {
	u := fmt.Sprintf("%s/environments/%s/featurestates/%d/", p.BaseURL, url.PathEscape(envKey), stateID)
	body := map[string]any{"enabled": enabled, "feature_state_value": strconv.Itoa(percentage)}
	if err := p.do(ctx, http.MethodPatch, u, body, nil); err != nil {
		return fmt.Errorf("flagsmith: update feature state: %w", err)
	}
	return nil
}

func (p Provider) do(ctx context.Context, method, u string, body, out any) error {
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+p.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
