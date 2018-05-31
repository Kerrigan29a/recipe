package recipe

import (
	"fmt"
	"os"
	"os/exec"
)

func IsSyscallKill(err error) bool {
	switch err := err.(type) {
	case *os.SyscallError:
		s := err.Syscall
		fmt.Printf("DEBUG SyscallError: %v\n", s)
		return true
	case *exec.ExitError:
		fmt.Printf("DEBUG ExitError: %v\n", err.Error())
		return true
	}
	fmt.Printf("DEBUG ErrorMessage = %v\n", err.Error())
	return false
}
