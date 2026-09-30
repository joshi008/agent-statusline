package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StableBinary prefers the PATH entry (e.g. /opt/homebrew/bin/agent-statusline, which survives
// `brew upgrade`) over the resolved Cellar path of the running executable.
func StableBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeReal, err := filepath.EvalSymlinks(exe)
	if err != nil {
		exeReal = exe
	}
	if lp, err := exec.LookPath(Marker); err == nil {
		if abs, err := filepath.Abs(lp); err == nil {
			if real, err := filepath.EvalSymlinks(abs); err == nil && real == exeReal {
				return abs, nil
			}
		}
	}
	return exeReal, nil
}

// CheckBinary refuses paths that will disappear (go run / temp dirs) or are not absolute.
func CheckBinary(path string) error {
	tmp := os.TempDir()
	if real, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = real
	}
	return checkBinary(path, tmp)
}

func checkBinary(path, tmp string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("binary path %q is not absolute", path)
	}
	tmp = filepath.Clean(tmp)
	if strings.Contains(path, "go-build") || strings.HasPrefix(path, tmp+string(filepath.Separator)) {
		return fmt.Errorf("binary %s is in a temporary directory (go run?); install it first with `go install` or Homebrew", path)
	}
	return nil
}
