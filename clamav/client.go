package clamav

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// Scanner 接口定义了防病毒扫描器的行为
type Scanner interface {
	ScanFile(filePath string) (string, error)
	GetVersion() (string, error)
	Ping() error
	Reload() error
	Shutdown() error
}

// Client 结构体表示ClamAV客户端
type Client struct {
	address string
}

// NewClient 创建一个新的ClamAV客户端
func NewClient(address string) Scanner {
	return &Client{address: address}
}

// ScanFile 扫描指定文件路径的文件
func (c *Client) ScanFile(filePath string) (string, error) {
	// 连接到ClamAV守护进程
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return "", fmt.Errorf("连接ClamAV失败: %v", err)
	}
	defer conn.Close()

	// 发送扫描命令
	_, err = fmt.Fprintf(conn, "SCAN %s\n", filePath)
	if err != nil {
		return "", fmt.Errorf("发送扫描命令失败: %v", err)
	}

	// 读取扫描结果
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	result := scanner.Text()

	// 解析扫描结果
	parts := strings.Split(result, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("无效的扫描结果: %s", result)
	}

	status := strings.TrimSpace(parts[1])
	return status, nil
}

// GetVersion 获取ClamAV版本信息
func (c *Client) GetVersion() (string, error) {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return "", fmt.Errorf("连接ClamAV失败: %v", err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "VERSION\n")
	if err != nil {
		return "", fmt.Errorf("发送版本命令失败: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	version := scanner.Text()

	return version, nil
}

// Ping 检查clamd是否正在运行
func (c *Client) Ping() error {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return fmt.Errorf("连接ClamAV失败: %v", err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "PING\n")
	if err != nil {
		return fmt.Errorf("发送PING命令失败: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	response := scanner.Text()

	if response != "PONG" {
		return fmt.Errorf("未收到预期的PONG响应: %s", response)
	}

	return nil
}

// Reload 重新加载病毒数据库
func (c *Client) Reload() error {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return fmt.Errorf("连接ClamAV失败: %v", err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "RELOAD\n")
	if err != nil {
		return fmt.Errorf("发送RELOAD命令失败: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	response := scanner.Text()

	if response != "RELOADING" {
		return fmt.Errorf("未收到预期的RELOADING响应: %s", response)
	}

	return nil
}

// Shutdown 关闭clamd服务
func (c *Client) Shutdown() error {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return fmt.Errorf("连接ClamAV失败: %v", err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "SHUTDOWN\n")
	if err != nil {
		return fmt.Errorf("发送SHUTDOWN命令失败: %v", err)
	}

	return nil
}
