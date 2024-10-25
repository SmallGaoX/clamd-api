package main

import (
	"github.com/SmallGaoX/clamd-api/cmd"
	"log"
	"os"
)

func main() {
	// 获取日志文件句柄
	logFile := os.Stderr
	if f, ok := log.Writer().(*os.File); ok {
		logFile = f
	}

	// 确保在程序退出时关闭日志文件
	defer logFile.Close()

	cmd.Execute()
}
