// internal/graph/graph.go
package graph

import (
	"encoding/json"
	"os"
)

// ProcessProject runs the codegraph analysis and writes out JSON.
func ProcessProject(projectPath, projectName, outputFile string) error {
	result, err := processGoProject(projectPath, projectName)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputFile, data, 0644)
}
