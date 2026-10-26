package main

import (
	"io"
	"os"

	"github.com/Lutherwaves/builder/internal/mcpmerge"
)

func init() { commands["mcp-merge"] = mcpMergeCmd }

// mcpMergeCmd prints the merged file. On any failure it prints nothing and exits 1: setup.sh then leaves the file unchanged.
func mcpMergeCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		io.WriteString(stderr, "usage: builder mcp-merge <.mcp.json> <server binary>\n")
		return 2
	}
	src, err := os.ReadFile(args[0])
	if err != nil {
		return 1
	}
	out, err := mcpmerge.Merge(src, args[1])
	if err != nil {
		return 1
	}
	stdout.Write(out)
	return 0
}
