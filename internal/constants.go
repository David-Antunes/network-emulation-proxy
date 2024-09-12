package internal

import (
	"fmt"
	"syscall"
)

const QUEUE_SIZE = 1000

func ShutdownAndLog(err error) {
	fmt.Println(err)
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
}
