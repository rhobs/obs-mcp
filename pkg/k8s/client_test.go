package k8s

import (
	"testing"
)

func TestGetRouteURLParseHost(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantHost string
		wantErr  bool
	}{
		{
			name:     "valid route with host",
			body:     `{"kind":"Route","spec":{"host":"thanos-querier.apps.example.com"}}`,
			wantHost: "https://thanos-querier.apps.example.com",
			wantErr:  false,
		},
		{
			name:     "route without host field",
			body:     `{"kind":"Route","spec":{}}`,
			wantHost: "",
			wantErr:  true,
		},
		{
			name:     "empty body",
			body:     `{}`,
			wantHost: "",
			wantErr:  true,
		},
		{
			name:     "host with port in URL",
			body:     `{"spec":{"host":"thanos-querier.apps.example.com:9091"}}`,
			wantHost: "https://thanos-querier.apps.example.com:9091",
			wantErr:  false,
		},
		{
			name:     "empty host value",
			body:     `{"spec":{"host":""}}`,
			wantHost: "",
			wantErr:  true,
		},
		{
			name:     "malformed JSON with host-like string",
			body:     `not json but has "host": in it`,
			wantHost: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := parseHostFromRouteBody(tt.body)
			if tt.wantErr && host != "" {
				t.Errorf("expected empty host, got %s", host)
			}
			if !tt.wantErr && host != tt.wantHost {
				t.Errorf("expected host %s, got %s", tt.wantHost, host)
			}
		})
	}
}
