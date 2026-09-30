//go:build unix

package enrich

import (
	"os/exec"
	"syscall"
)

// killGroupOnCancel runs cmd in its own process group and makes cancellation kill the whole
// group, so a compound command's grandchildren (which hold stdout open) die with it.
func killGroupOnCancel(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
