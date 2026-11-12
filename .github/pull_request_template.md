## Summary

<!-- What does this PR change and why? -->

## Contribution check

Observability **toolsets / handlers / `pkg/obs` clients live in
[kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server)**.
obs-mcp is a thin standalone host. See [CONTRIBUTING.md](https://github.com/rhobs/obs-mcp/blob/main/CONTRIBUTING.md).

- [ ] This PR is **host-related** (CLI, adapter, manifests, evals, docs, pin bump), **or**
- [ ] Toolset/backend logic already landed upstream and this PR only bumps / adapts obs-mcp

## Test plan

- [ ] `make test-unit`
- [ ] `make generate-tools-doc` / `TOOLS.md` updated if the kube-mcp pin changed tool docs
- [ ] e2e / manual checks (if applicable):
