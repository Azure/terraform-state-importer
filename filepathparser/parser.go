package filepathparser

import (
	"os"
	"path/filepath"
	"strings"
)

func ParsePath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		dirname, _ := os.UserHomeDir()
		path = filepath.Join(dirname, path[2:])
	}

	return filepath.Abs(path)
}
