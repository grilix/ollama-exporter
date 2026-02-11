package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	sha     = "unknown"
	buildTm = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the version information for ollama-exporter.`,
	Run: func(cmd *cobra.Command, args []string) {
		versionString := GetVersion()
		fmt.Printf("ollama-exporter version %s\n", versionString)
		fmt.Printf("Built at: %s\n", buildTm)
	},
}

func GetVersion() string {
	versionString := version
	if versionString != "dev" && versionString != "main" {
		return versionString
	}

	shaPrefix := sha
	if len(shaPrefix) == 40 {
		shaPrefix = shaPrefix[:7]
	}
	return fmt.Sprintf("%s:%s", versionString, shaPrefix)
}
