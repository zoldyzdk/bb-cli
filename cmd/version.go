package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the CLI version, Go toolchain version, and VCS metadata when embedded by the build.`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("bb: build information not available")
		return
	}

	ver := bi.Main.Version
	if ver == "" {
		ver = "unknown"
	}
	fmt.Printf("bb %s\n", ver)
	fmt.Printf("go %s\n", bi.GoVersion)

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			fmt.Printf("commit %s\n", s.Value)
		case "vcs.time":
			fmt.Printf("time %s\n", s.Value)
		}
	}
}
