package mcp

import (
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"k8s.io/utils/ptr"
)

// isToolApplicable matches kubernetes-mcp-server's ReadOnly / DisableDestructive
// filter. Host embedding still filters in kubernetes-mcp; this is for standalone.
func isToolApplicable(tool api.ServerTool, readOnly, disableDestructive bool) bool {
	if readOnly && !ptr.Deref(tool.Tool.Annotations.ReadOnlyHint, false) {
		return false
	}
	if disableDestructive && ptr.Deref(tool.Tool.Annotations.DestructiveHint, false) {
		return false
	}
	return true
}
