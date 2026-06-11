# rollops-plugin-flagsmith

A [Rollops](https://github.com/klarlabs-studio/rollops) feature-flag provider
plugin backed by [Flagsmith](https://flagsmith.com). As a rollout progresses,
this plugin drives a Flagsmith flag's enabled state and rollout percentage to
match — so a Flagsmith flag tracks a Rollops canary in lockstep.

It is a standalone binary that speaks the Rollops plugin protocol (manifest +
gRPC, `featureflag` capability). Rollops launches it as a subprocess; it never
needs to run as a service.

## Install

```bash
go install github.com/klarlabs-studio/rollops-plugin-flagsmith/cmd/rollops-plugin-flagsmith@latest
# or build from source:
make build && make checksum   # prints the sha256 to pin
```

Place the binary somewhere the Rollops daemon can read but cannot be tampered
with (a root-owned dir on a trusted mount — see the Rollops plugin security
model), and note its sha256.

## Configure

The plugin reads its Flagsmith credentials from **its own environment** —
Rollops passes only the flag name, environment, and percentage, never secrets:

| Env var | Meaning |
|---|---|
| `FLAGSMITH_API_URL` | Admin API base (default `https://api.flagsmith.com/api/v1`) |
| `FLAGSMITH_TOKEN` | Flagsmith Admin API token (required) |
| `FLAGSMITH_PROJECT_ID` | Numeric project id, for feature lookup (required) |

Then reference it from a rollout:

```yaml
spec:
  strategy:
    type: canary
    steps:
      - weight: 25
      - weight: 50
  featureFlags:
    plugin: /usr/local/lib/rollops/plugins/rollops-plugin-flagsmith
    sha256: <paste from `make checksum`>
    flag: checkout-redesign
    environment: <your Flagsmith environment API key>
    when: both
```

`environment` is the Flagsmith environment's API key. As the canary steps, the
flag's value is set to the current percentage (`25`, `50`, `100`) and `enabled`
follows the rollout; on promote it settles at 100%.

## Semantics

Per `ApplyFlag(flag, environment, percentage, disabled)` the plugin:

1. resolves the feature id by name within `FLAGSMITH_PROJECT_ID`;
2. resolves the environment's feature-state id;
3. PATCHes it: `enabled = !disabled`, `feature_state_value = percentage`.

Your application reads the flag's value to gate on the current rollout
percentage. (Flagsmith multivariate split is out of scope for v1; the value
carries the percentage, which is the simplest provider-correct mapping.)

## License

MIT.
