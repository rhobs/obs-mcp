package alertmanagement

// ServerPrompt provides instructions for LLMs using the alert management toolset.
const ServerPrompt = `
## Alert Rule Management Workflow

This toolset manages OpenShift alert rules via the monitoring-plugin management API.
It supports both in-console (Lightspeed) and non-console (Cursor, Claude, CLI) usage.

When managing alert rules:
1. List existing rules first to understand the current state
2. Respect RBAC boundaries — operations are scoped to the user's permissions
3. Do not attempt to modify GitOps-managed or operator-managed rules
4. Use standard severity levels: critical, warning, info, none
5. Provide clear PromQL expressions — validate syntax before creating rules`
