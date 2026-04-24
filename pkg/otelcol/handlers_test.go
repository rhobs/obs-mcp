package otelcol

import (
	"context"
	"testing"

	"github.com/pavolloffay/opentelemetry-mcp-server/modules/collectorschema"
)

// MockSchemaLoader is a mock implementation of SchemaLoader for testing.
type MockSchemaLoader struct {
	versions    []string
	latestVer   string
	components  map[collectorschema.ComponentType][]string
	schema      *collectorschema.ComponentSchema
	validateErr error
	validateRes *ValidationResult
}

func (m *MockSchemaLoader) GetComponentSchema(componentType collectorschema.ComponentType, componentName string, version string) (*collectorschema.ComponentSchema, error) {
	if m.schema != nil {
		return m.schema, nil
	}
	return &collectorschema.ComponentSchema{
		Name:    componentName,
		Type:    componentType,
		Version: version,
		Schema:  map[string]interface{}{"type": "object"},
	}, nil
}

func (m *MockSchemaLoader) GetComponentSchemaJSON(componentType collectorschema.ComponentType, componentName string, version string) ([]byte, error) {
	return []byte(`{"type":"object"}`), nil
}

func (m *MockSchemaLoader) ListAvailableComponents(version string) (map[collectorschema.ComponentType][]string, error) {
	if m.components != nil {
		return m.components, nil
	}
	return map[collectorschema.ComponentType][]string{
		collectorschema.ComponentTypeReceiver:  {"otlp", "prometheus"},
		collectorschema.ComponentTypeProcessor: {"batch", "memory_limiter"},
		collectorschema.ComponentTypeExporter:  {"otlp", "debug"},
		collectorschema.ComponentTypeExtension: {"health_check"},
		collectorschema.ComponentTypeConnector: {"forward"},
	}, nil
}

func (m *MockSchemaLoader) ValidateComponentYAML(componentType collectorschema.ComponentType, componentName string, version string, yamlData []byte) (*ValidationResult, error) {
	if m.validateErr != nil {
		return nil, m.validateErr
	}
	if m.validateRes != nil {
		return m.validateRes, nil
	}
	return &ValidationResult{Valid: true}, nil
}

func (m *MockSchemaLoader) ValidateComponentJSON(componentType collectorschema.ComponentType, componentName string, version string, jsonData []byte) (*ValidationResult, error) {
	if m.validateErr != nil {
		return nil, m.validateErr
	}
	if m.validateRes != nil {
		return m.validateRes, nil
	}
	return &ValidationResult{Valid: true}, nil
}

func (m *MockSchemaLoader) GetLatestVersion() (string, error) {
	if m.latestVer != "" {
		return m.latestVer, nil
	}
	return "v0.100.0", nil
}

func (m *MockSchemaLoader) GetAllVersions() ([]string, error) {
	if m.versions != nil {
		return m.versions, nil
	}
	return []string{"v0.100.0", "v0.99.0", "v0.98.0"}, nil
}

func TestListComponentsHandler(t *testing.T) {
	loader := &MockSchemaLoader{}
	ctx := context.Background()

	input := ListComponentsInput{Version: "v0.100.0"}
	result := ListComponentsHandler(ctx, loader, input)

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}

func TestGetComponentSchemaHandler(t *testing.T) {
	loader := &MockSchemaLoader{}
	ctx := context.Background()

	tests := []struct {
		name    string
		input   GetComponentSchemaInput
		wantErr bool
	}{
		{
			name: "valid receiver",
			input: GetComponentSchemaInput{
				ComponentType: ComponentTypeReceiver,
				ComponentName: "otlp",
				Version:       "v0.100.0",
			},
			wantErr: false,
		},
		{
			name: "invalid component type",
			input: GetComponentSchemaInput{
				ComponentType: "invalid",
				ComponentName: "otlp",
			},
			wantErr: true,
		},
		{
			name: "missing component name",
			input: GetComponentSchemaInput{
				ComponentType: ComponentTypeReceiver,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetComponentSchemaHandler(ctx, loader, tt.input)
			if (result.Error != nil) != tt.wantErr {
				t.Errorf("GetComponentSchemaHandler() error = %v, wantErr %v", result.Error, tt.wantErr)
			}
		})
	}
}

func TestValidateConfigHandler(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		loader  *MockSchemaLoader
		input   ValidateConfigInput
		wantErr bool
	}{
		{
			name:   "valid yaml config",
			loader: &MockSchemaLoader{validateRes: &ValidationResult{Valid: true}},
			input: ValidateConfigInput{
				ComponentType: ComponentTypeReceiver,
				ComponentName: "otlp",
				Config:        "protocols:\n  grpc:\n    endpoint: 0.0.0.0:4317",
				Format:        "yaml",
			},
			wantErr: false,
		},
		{
			name:   "invalid format",
			loader: &MockSchemaLoader{},
			input: ValidateConfigInput{
				ComponentType: ComponentTypeReceiver,
				ComponentName: "otlp",
				Config:        "{}",
				Format:        "xml",
			},
			wantErr: true,
		},
		{
			name:   "missing config",
			loader: &MockSchemaLoader{},
			input: ValidateConfigInput{
				ComponentType: ComponentTypeReceiver,
				ComponentName: "otlp",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateConfigHandler(ctx, tt.loader, tt.input)
			if (result.Error != nil) != tt.wantErr {
				t.Errorf("ValidateConfigHandler() error = %v, wantErr %v", result.Error, tt.wantErr)
			}
		})
	}
}

func TestGetVersionsHandler(t *testing.T) {
	loader := &MockSchemaLoader{
		versions:  []string{"v0.100.0", "v0.99.0"},
		latestVer: "v0.100.0",
	}
	ctx := context.Background()

	result := GetVersionsHandler(ctx, loader, GetVersionsInput{})

	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}

func TestComponentType_IsValid(t *testing.T) {
	tests := []struct {
		ct   ComponentType
		want bool
	}{
		{ComponentTypeReceiver, true},
		{ComponentTypeProcessor, true},
		{ComponentTypeExporter, true},
		{ComponentTypeExtension, true},
		{ComponentTypeConnector, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.ct), func(t *testing.T) {
			if got := tt.ct.IsValid(); got != tt.want {
				t.Errorf("ComponentType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
