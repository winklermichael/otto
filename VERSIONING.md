# Versioning

Otto uses [Semantic Versioning 2.0.0](https://semver.org/) for both the application and the Helm chart.  
Versioning is handled independently for the application and the Helm chart to allow flexibility in releasing changes.

---

## Version Types

### Application Version (`vMAJOR.MINOR.PATCH`)
- Tracked via Git **tags**: `v0.4.0`, `v1.0.0`, etc.
- Controls the version of the compiled Go operator
- Updated in Helm chart as `appVersion`

### Helm Chart Version (`Chart.yaml` → `version`)
- Has its own versioning independent of the app
- Bump when:
  - Templates or `values.yaml` change
  - Metadata updates (labels, annotations, README)
  - Chart logic changes even if the app code stays the same

### `appVersion` (inside Chart.yaml)
- Tracks the **Go application version**
- Always matches a real application Git tag (e.g., `v0.4.1`)

---

## Branching Flow

Otto follows a **Git Flow-inspired model**:

### Primary Branches
- `main`: stable, production-ready releases
- `dev`: integration branch where active development happens

### Supporting Branches
- `feature/*`: new features, branched off `dev`
- `fix/*`: bug fixes, also branched off `dev`
- `release/*`: optional, for staging a release from `dev`
- `hotfix/*`: critical fixes based on `main`

> All PRs are merged into `dev` first. When ready, `dev` is merged into `main` and a release is tagged.

---

## Versioning Policy

| Use Case | App Version | Chart Version | Chart `appVersion` | Example Git Tag |
|----------|-------------|----------------|--------------------|------------------|
| App only | ✅ bump      | ❌ unchanged    | ✅ bump            | `v0.4.1`         |
| Chart only | ❌ unchanged | ✅ bump        | ❌ unchanged        | `chart-v0.3.0`   |
| Both     | ✅ bump      | ✅ bump         | ✅ bump            | `v0.5.0`         |

---

## Release Process

1. Finish feature/fix branches and merge into `dev`
2. When `dev` is stable, create a PR into `main`
3. Tag a release on `main`:
   - App release: `vX.Y.Z`
   - Chart-only release: `chart-vX.Y.Z`
4. GitHub Actions will:
   - Build and release the Go binary
   - Package and publish the Helm chart via `chart-releaser`

---

## Examples

- A bug fix in the app:
  - Tag: `v0.4.1`
  - `Chart.yaml`: `version: 0.3.0`, `appVersion: v0.4.1`

- A Helm chart change (new values, better templates):
  - Tag: `chart-v0.3.1`
  - `Chart.yaml`: `version: 0.3.1`, `appVersion: v0.4.1` (unchanged app)

- A full new feature (OAuth provider + Helm values):
  - Tag: `v0.5.0`
  - `Chart.yaml`: `version: 0.4.0`, `appVersion: v0.5.0`

---

## Tools

- Go app versioning: Git tags prefixed with `v`
- Helm chart versioning: Managed in `Chart.yaml`
- Releases published automatically via GitHub Actions and chart-releaser

