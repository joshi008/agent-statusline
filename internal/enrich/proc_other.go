//go:build !unix

package enrich

import "os/exec"

// killGroupOnCancel keeps exec's default cancellation (kill the process) where there are no
// unix process groups; cmd.WaitDelay still bounds how long Output waits on inherited pipes.
func killGroupOnCancel(cmd *exec.Cmd) {}
