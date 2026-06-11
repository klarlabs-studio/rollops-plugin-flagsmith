package flagsmith

import "os"

// FromEnv builds a Provider from the plugin's environment. Secrets and endpoint
// come from the plugin process, never from the Rollops target spec (Rollops
// passes only the flag name, environment, and percentage).
//
//	FLAGSMITH_API_URL     base Admin API URL (default https://api.flagsmith.com/api/v1)
//	FLAGSMITH_TOKEN       Admin API token (required)
//	FLAGSMITH_PROJECT_ID  numeric project id (required)
func FromEnv() Provider {
	base := os.Getenv("FLAGSMITH_API_URL")
	if base == "" {
		base = "https://api.flagsmith.com/api/v1"
	}
	return Provider{
		BaseURL:   base,
		Token:     os.Getenv("FLAGSMITH_TOKEN"),
		ProjectID: os.Getenv("FLAGSMITH_PROJECT_ID"),
	}
}
