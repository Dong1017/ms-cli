package fs

import "github.com/vigo999/ms-cli/internal/workspacefile"

func resolveSafePath(workDir, input string) (string, error) {
	return workspacefile.ResolvePath(workDir, input)
}
