package internal

import (
	"fmt"
	"runtime/debug"
	"syscall"
)

const QUEUE_SIZE = 10000

func ShutdownAndLog(err error) {
	fmt.Println(err)
	debug.PrintStack()
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
}
