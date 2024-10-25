package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SmallGaoX/clamd-api/cmd"
	"github.com/SmallGaoX/clamd-api/version"
)

func main() {
	// 打印版本信息
	fmt.Printf("版本: %s\n", version.Version)
	fmt.Printf("提交SHA: %s\n", version.CommitSHA)
	fmt.Printf("构建时间: %s\n", version.BuildTime)

	// 获取日志文件句柄
	logFile := os.Stderr
	if f, ok := log.Writer().(*os.File); ok {
		logFile = f
	}

	// 确保在程序退出时关闭日志文件
	defer logFile.Close()

	cmd.Execute()
}
