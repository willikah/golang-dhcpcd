package cmd

import (
	"fmt"
	"golang-dhcpcd/internal/pkg/version"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and git info",
	Run: func(cmd *cobra.Command, args []string) {
		info := version.GetGitInfo()
		fmt.Printf("Tag: %s\nBranch: %s\nCommit: %s\nDirty: %v\n", info.Tag, info.Branch, info.Commit, info.Dirty)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
