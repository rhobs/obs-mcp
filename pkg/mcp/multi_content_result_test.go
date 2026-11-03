package mcp

import (
	"errors"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestNewMultiContentResult(t *testing.T) {
	t.Run("text and image blocks", func(t *testing.T) {
		png := []byte{0x89, 0x50, 0x4e, 0x47}
		result := NewMultiContentResult([]*api.ToolContent{
			api.NewTextToolContent("boot prompt"),
			api.NewImageToolContent(png, "image/png"),
		}, map[string]any{"ok": true}, nil)

		require.False(t, result.IsError)
		require.Equal(t, map[string]any{"ok": true}, result.StructuredContent)
		require.Len(t, result.Content, 2)

		text, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		require.Equal(t, "boot prompt", text.Text)

		image, ok := result.Content[1].(*mcp.ImageContent)
		require.True(t, ok)
		require.Equal(t, "image/png", image.MIMEType)
		require.Equal(t, png, image.Data)
	})

	t.Run("error path", func(t *testing.T) {
		result := NewMultiContentResult(nil, nil, errors.New("sample"))
		require.True(t, result.IsError)
		require.Len(t, result.Content, 1)
		text, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		require.Equal(t, "sample", text.Text)
	})

	t.Run("image without data", func(t *testing.T) {
		result := NewMultiContentResult([]*api.ToolContent{
			{MIMEType: "image/png"},
		}, nil, nil)
		require.True(t, result.IsError)
	})
}
