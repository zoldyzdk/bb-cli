package cmd

import (
	"fmt"
	"io"
	"runtime/debug"
	"strings"

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
	rootCmd.Version = buildInfoText()
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}

func runVersion(cmd *cobra.Command, args []string) {
	writeBuildInfo(cmd.OutOrStdout())
}

func writeBuildInfo(w io.Writer) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Fprintln(w, "bb: build information not available")
		return
	}

	ver := bi.Main.Version
	if ver == "" {
		ver = "unknown"
	}
	fmt.Fprintf(w, "bb %s\n", ver)
	fmt.Fprintf(w, "go %s\n", bi.GoVersion)

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			fmt.Fprintf(w, "commit %s\n", s.Value)
		case "vcs.time":
			fmt.Fprintf(w, "time %s\n", s.Value)
		}
	}
}

func buildInfoText() string {
	var b strings.Builder
	writeBuildInfo(&b)
	return strings.TrimSuffix(b.String(), "\n")
}
