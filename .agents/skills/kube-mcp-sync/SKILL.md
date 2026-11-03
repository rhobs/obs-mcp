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

Observability tool definitions live upstream under
`pkg/toolsets/{metrics,logs,traces,otelcol}` and `pkg/obs/`.
obs-mcp only hosts/adapts them (`pkg/mcp`, `pkg/clusterinspector`).

## 1. Bump kubernetes-mcp-server

Record the current pin, then update to latest `main` and tidy:

```bash
set -euo pipefail
OLD=$(go list -m -f '{{.Version}}' github.com/containers/kubernetes-mcp-server)
go get github.com/containers/kubernetes-mcp-server@main
go mod tidy
NEW=$(go list -m -f '{{.Version}}' github.com/containers/kubernetes-mcp-server)
MOD=$(go list -m -f '{{.Dir}}' github.com/containers/kubernetes-mcp-server)
echo "bumped $OLD → $NEW"
```

Ask before applying this bump if `go.mod` already has uncommitted changes.
If the bump fails, stop — do not run audits against the old module.
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

## 3. Adapter / host parity

Tool definitions are no longer authored in obs-mcp. Audit whether the
**standalone adapter and inspector** keep up with upstream `ServerTool` /
`ClusterInspector` / result APIs:

```bash
# Upstream adapter + result helpers the fork must track
rg -n 'RBACMetadataKey|ContentBlocks|NewMultiContentResult|BearerToken' \
  "$MOD/pkg/mcp/tools_gosdk.go" "$MOD/pkg/mcp/multi_content_result.go" "$MOD/pkg/api/"

# Local adapter / registration / inspector
rg -n 'RBACMetadataKey|ContentBlocks|NewMultiContentResult|BearerToken|ServerToolToGoSdkTool|addToolset' \
  --glob '*.go' pkg/mcp pkg/clusterinspector

# Diff key adapter files against upstream for missed merges
diff -u "$MOD/pkg/mcp/tools_gosdk.go" pkg/mcp/tools_gosdk.go || true
diff -u "$MOD/pkg/mcp/multi_content_result.go" pkg/mcp/multi_content_result.go || true
```

Flag gaps such as:

- `ServerTool.RBAC` not merged into tool `Meta`
- `ToolCallResult.ContentBlocks` not converted
- New `ClusterInspector` methods missing on `pkg/clusterinspector`
- New `api.Toolset` methods (`GetPrompts` / `GetResources` / …) the host ignores

Also regenerate docs after a successful bump:

```bash
make generate-tools-doc
make check-tools-doc
```

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
