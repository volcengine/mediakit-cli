package main

import (
	"errors"
	"fmt"
	"os"

	"mediakit-cli/internal/cliexit"
	"mediakit-cli/internal/commands"
)

func main() {
	err := commands.Execute()
	if err == nil {
		return
	}
	// 业务失败：结构化错误已写入 stdout JSON，stderr 不重复打印
	if !errors.Is(err, cliexit.ErrBusinessFailure) {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
