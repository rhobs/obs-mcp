---
name: kube-mcp-sync
description: >-
  Bump kubernetes-mcp-server, then audit obs-mcp for deprecated APIs
  and shared-dependency mismatches. Invoke explicitly only.
disable-model-invocation: true
---

# kube-mcp Upstream Sync

Bump upstream → audit → report → ask before each fix → verify
(`go build ./...`, `go test ./...`) → ask before commit
(`git commit -s -S`, conventional commit).

## 1. Bump kubernetes-mcp-server

Record the current pin, then update to latest `main` and tidy:

```bash
OLD=$(go list -m -f '{{.Version}}' github.com/containers/kubernetes-mcp-server)
go get github.com/containers/kubernetes-mcp-server@main
go mod tidy
NEW=$(go list -m -f '{{.Version}}' github.com/containers/kubernetes-mcp-server)
MOD=$(go list -m -f '{{.Dir}}' github.com/containers/kubernetes-mcp-server)
echo "bumped $OLD → $NEW"
```

Ask before applying this bump if `go.mod` already has uncommitted changes.
If `$OLD == $NEW`, still run the audits below (pin may be current but code may lag).

## 2. Deprecated / removed APIs

Against the **new** module (`$MOD`):

1. Find every upstream `Deprecated:` annotation and extract the symbol that follows
   (type, func, method, or alias declaration after the comment).
2. For **each** extracted symbol, search obs-mcp call sites — do not rely on a
   hard-coded list.
3. Run `go build ./...` — compile errors are the primary signal for removed APIs.

```bash
# Upstream deprecations across the whole module (api, config, kubernetes, toolsets, …)
rg -n -B2 -A5 'Deprecated:' "$MOD"

# For each deprecated symbol NAME found above, search local usage, e.g.:
#   rg -n '\bExtendedConfig\b|\bOptionalString\b|\bBaseConfig\b' --glob '*.go'
# Build the pattern dynamically from all extracted names — not a fixed list.
```

Report each obs-mcp hit with file:line and the upstream-suggested replacement.

## 3. New struct fields / conventions

Upstream may add new fields to types obs-mcp uses (e.g. `RBAC` on `ServerTool`).
These don't break the build but obs-mcp should adopt them.

Check key types for fields obs-mcp never sets:

```bash
# Extract fields from upstream ServerTool and ServerPrompt structs
rg 'type (ServerTool|ServerPrompt) struct' -A30 "$MOD/pkg/api/"

# Check which fields obs-mcp actually populates
rg 'ServerTool\{' --glob '*.go' -A15 pkg/
rg 'ServerPrompt\{' --glob '*.go' -A15 pkg/
```

For each field defined upstream, search obs-mcp for usage. Flag any field
that appears in zero obs-mcp definitions — it may be a new convention to adopt.
Cross-check with upstream in-tree toolsets to confirm it's expected:

```bash
# How upstream toolsets set this field (e.g. RBAC, Annotations, etc.)
rg 'RBAC:|Annotations:' "$MOD/pkg/" --glob '*.go' | head -20
```

Report each missing field with the upstream pattern to follow.

## 4. Shared dependencies

```bash
go mod edit -json go.mod      | jq -r '.Require[]? | "\(.Path) \(.Version)"' | sort > /tmp/obs-deps.txt
go mod edit -json "$MOD/go.mod" | jq -r '.Require[]? | "\(.Path) \(.Version)"' | sort > /tmp/upstream-deps.txt
join /tmp/obs-deps.txt /tmp/upstream-deps.txt | awk '$2 != $3 { print $1, $2, $3 }'
```

Classify mismatches:

- **Action required**: obs-mcp behind; or critical deps that must match kube-mcp
  even when kube-mcp is lower (at least `github.com/modelcontextprotocol/go-sdk`)
- **Informational**: obs-mcp ahead on non-critical deps
