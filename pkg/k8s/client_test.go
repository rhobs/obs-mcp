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
		{
			name:     "host in wrong JSON location - should only parse spec.host",
			body:     `{"status":{"host":"wrong-host.com"},"spec":{"host":"correct.example.com"}}`,
			wantHost: "https://correct.example.com",
			wantErr:  false,
		},
		{
			name:     "host pattern in annotation should not be parsed",
			body:     `{"metadata":{"annotations":{"config":"host\":\"invalid.url.com"}},"spec":{"host":"real.example.com"}}`,
			wantHost: "https://real.example.com",
			wantErr:  false,
		},
		{
			name:     "nested host field should not confuse parser",
			body:     `{"spec":{"tls":{"host":"tls-host.com"},"host":"correct.example.com"}}`,
			wantHost: "https://correct.example.com",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, err := parseHostFromRouteBody([]byte(tt.body))
			if tt.wantErr {
				if host != "" {
					t.Errorf("expected empty host, got %s", host)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if host != tt.wantHost {
					t.Errorf("expected host %s, got %s", tt.wantHost, host)
				}
			}
		})
	}
}
