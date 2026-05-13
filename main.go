package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("sfsAI —— 边缘 AI 时代的通用数据底座")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run ./cmd/sfsai    启动 Sidecar 服务")
	fmt.Println("  go run ./cmd/sfsai -h 查看帮助")
	fmt.Println()
	fmt.Println("快速启动:")
	fmt.Println("  go run ./cmd/sfsai -db ./sfsai_data -addr :8630")
	fmt.Println()

	cmd := exec.Command("go", "run", "./cmd/sfsai")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("启动失败: %v\n", err)
		os.Exit(1)
	}
}
