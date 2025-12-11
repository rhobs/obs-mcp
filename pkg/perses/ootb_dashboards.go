package perses

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// OOTBDashboardsConfig represents the YAML structure for out-of-the-box dashboards
type OOTBDashboardsConfig struct {
	Dashboards []PersesDashboardInfo `yaml:"dashboards"`
}

// LoadOOTBDashboards loads out-of-the-box dashboard definitions from a YAML file
func LoadOOTBDashboards(filePath string) ([]PersesDashboardInfo, error) {
	if filePath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read OOTB dashboards file: %w", err)
	}

	var config OOTBDashboardsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse OOTB dashboards YAML: %w", err)
	}

	return config.Dashboards, nil
}
