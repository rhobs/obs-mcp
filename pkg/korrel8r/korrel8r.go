// Package korrel8r provides an MCP forwarding proxy to a remote korrel8r MCP server.
// It discovers tools dynamically at startup and forwards all calls transparently,
// preserving instructions, tool descriptions, and schemas identically.
package korrel8r

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	Endpoint string
	Insecure bool
}

// Proxy connects to a remote korrel8r MCP server and forwards tool calls to it.
type Proxy struct {
	cs           *mcp.ClientSession
	tools        []*mcp.Tool
	instructions string
}

// Connect creates a proxy by connecting to the remote korrel8r MCP server,
// discovering its tools, and capturing its instructions.
func Connect(ctx context.Context, cfg *Config) (*Proxy, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}
	}
	httpClient := &http.Client{
		Transport: &authRoundTripper{next: transport},
	}

	client := mcp.NewClient(
		&mcp.Implementation{Name: "obs-mcp-korrel8r-proxy", Version: "1.0.0"},
		nil,
	)
	cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             cfg.Endpoint,
		HTTPClient:           httpClient,
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to korrel8r MCP server: %w", err)
	}

	tools, err := cs.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		_ = cs.Close()
		return nil, fmt.Errorf("listing korrel8r tools: %w", err)
	}

	return &Proxy{
		cs:           cs,
		tools:        tools.Tools,
		instructions: cs.InitializeResult().Instructions,
	}, nil
}

// Instructions returns the instructions from the remote korrel8r server.
func (p *Proxy) Instructions() string {
	return p.instructions
}

// AddTools registers all discovered tools on the MCP server with forwarding handlers.
func (p *Proxy) AddTools(mcpServer *mcp.Server) {
	for _, t := range p.tools {
		mcpServer.AddTool(t, forwardHandler(p.cs, t.Name))
	}
}

// Close closes the connection to the remote server.
func (p *Proxy) Close() {
	if p.cs != nil {
		_ = p.cs.Close()
	}
}

// forwardHandler returns a ToolHandler that forwards the call to the remote server.
func forwardHandler(cs *mcp.ClientSession, name string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if req.Extra != nil && req.Extra.Header != nil {
			if authHeader := req.Extra.Header.Get("Authorization"); authHeader != "" {
				if token, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
					ctx = withToken(ctx, token)
				}
			}
		}

		var args map[string]any
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return nil, err
			}
		}

		return cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		})
	}
}

type tokenKey struct{}

func withToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, tokenKey{}, token)
}

func tokenFromContext(ctx context.Context) string {
	token, _ := ctx.Value(tokenKey{}).(string)
	return token
}

// authRoundTripper forwards bearer tokens from context to outgoing HTTP requests.
type authRoundTripper struct {
	next http.RoundTripper
}

func (rt *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if token := tokenFromContext(req.Context()); token != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return rt.next.RoundTrip(req)
}
