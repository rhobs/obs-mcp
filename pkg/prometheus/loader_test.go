package prometheus

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/common/model"
)

func TestMakeLLMFriendlyError(t *testing.T) {
	tests := []struct {
		name           string
		originalError  error
		query          string
		expectedSubstr []string // substrings that should be in the error message
	}{
		{
			name:          "parse error",
			originalError: errors.New("parse error: unexpected character"),
			query:         "up{invalid",
			expectedSubstr: []string{
				"syntax error",
				"up{invalid",
				"check the query syntax",
			},
		},
		{
			name:          "bad_data error",
			originalError: errors.New("bad_data: invalid expression"),
			query:         "rate(http[5m])",
			expectedSubstr: []string{
				"syntax error",
				"rate(http[5m])",
				"correctly formatted",
			},
		},
		{
			name:          "unknown function",
			originalError: errors.New("unknown function: foobar"),
			query:         "foobar(up)",
			expectedSubstr: []string{
				"unknown function",
				"foobar(up)",
				"function name is correct",
			},
		},
		{
			name:          "timeout error",
			originalError: errors.New("query timeout exceeded"),
			query:         "sum(rate(http_requests_total[5m])) by (job)",
			expectedSubstr: []string{
				"timed out",
				"sum(rate(http_requests_total[5m])) by (job)",
				"time range",
				"step size",
			},
		},
		{
			name:          "deadline exceeded",
			originalError: errors.New("context deadline exceeded"),
			query:         "up",
			expectedSubstr: []string{
				"timed out",
				"up",
				"reducing the time range",
			},
		},
		{
			name:          "connection refused",
			originalError: errors.New("connection refused"),
			query:         "up",
			expectedSubstr: []string{
				"cannot connect",
				"Prometheus server is running",
			},
		},
		{
			name:          "no such host",
			originalError: errors.New("no such host: prometheus.example.com"),
			query:         "up",
			expectedSubstr: []string{
				"cannot connect",
				"Prometheus server is running",
			},
		},
		{
			name:          "generic error",
			originalError: errors.New("some other error"),
			query:         "up",
			expectedSubstr: []string{
				"up",
				"some other error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeLLMFriendlyError(tt.originalError, tt.query)
			if result == nil {
				t.Fatalf("expected error, got nil")
			}

			resultMsg := result.Error()
			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(resultMsg, substr) {
					t.Errorf("expected error to contain %q, got: %s", substr, resultMsg)
				}
			}
		})
	}
}

func TestMakeLLMFriendlyError_NilError(t *testing.T) {
	result := makeLLMFriendlyError(nil, "up")
	if result != nil {
		t.Errorf("expected nil error for nil input, got: %v", result)
	}
}

func TestCheckEmptyResult(t *testing.T) {
	tests := []struct {
		name           string
		result         any
		query          string
		expectWarning  bool
		expectedSubstr []string
	}{
		{
			name:          "empty matrix",
			result:        model.Matrix{},
			query:         "nonexistent_metric",
			expectWarning: true,
			expectedSubstr: []string{
				"nonexistent_metric",
				"returned no data",
				"metric does not exist",
				"no data for the specified time range",
				"list_metrics",
			},
		},
		{
			name:          "empty vector",
			result:        model.Vector{},
			query:         "up{job=\"missing\"}",
			expectWarning: true,
			expectedSubstr: []string{
				"up{job=\"missing\"}",
				"returned no data",
				"label filters are too restrictive",
			},
		},
		{
			name: "non-empty matrix",
			result: model.Matrix{
				&model.SampleStream{
					Metric: model.Metric{"__name__": "up"},
					Values: []model.SamplePair{{Timestamp: 0, Value: 1}},
				},
			},
			query:         "up",
			expectWarning: false,
		},
		{
			name: "non-empty vector",
			result: model.Vector{
				&model.Sample{
					Metric:    model.Metric{"__name__": "up"},
					Timestamp: 0,
					Value:     1,
				},
			},
			query:         "up",
			expectWarning: false,
		},
		{
			name:          "nil scalar",
			result:        (*model.Scalar)(nil),
			query:         "scalar(nonexistent)",
			expectWarning: true,
		},
		{
			name:          "valid scalar",
			result:        &model.Scalar{Value: 1, Timestamp: 0},
			query:         "scalar(up)",
			expectWarning: false,
		},
		{
			name:          "nil string",
			result:        (*model.String)(nil),
			query:         "string_metric",
			expectWarning: true,
		},
		{
			name:          "valid string",
			result:        &model.String{Value: "test", Timestamp: 0},
			query:         "string_metric",
			expectWarning: false,
		},
		{
			name:          "unknown type",
			result:        "unknown",
			query:         "up",
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning := checkEmptyResult(tt.result, tt.query)

			if tt.expectWarning {
				if warning == "" {
					t.Errorf("expected warning for empty result, got none")
				}
				for _, substr := range tt.expectedSubstr {
					if !strings.Contains(warning, substr) {
						t.Errorf("expected warning to contain %q, got: %s", substr, warning)
					}
				}
			} else {
				if warning != "" {
					t.Errorf("expected no warning, got: %s", warning)
				}
			}
		})
	}
}
