package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/srinidhi-metadome/go-codegraph-cli/internal/graph"
)

var (
	projectPath string
	projectName string
	outputFile  string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "codegraph",
	Short: "Analyze a Go project and produce a codegraph JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		return graph.ProcessProject(projectPath, projectName, outputFile)
	},
}

func init() {
	rootCmd.Flags().StringVarP(&projectPath, "path", "p", ".", "Go project root path")
	rootCmd.Flags().StringVarP(&projectName, "name", "n", "MyProject", "Project name in JSON")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "output.json", "Output JSON file")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
