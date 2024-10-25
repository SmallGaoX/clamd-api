package clamav

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
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
	ScanStream(data io.Reader) (string, error)
	MultiScan(filePaths []string) (map[string]string, error)
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

// ScanStream 扫描通过网络流传输的数据
func (c *Client) ScanStream(data io.Reader) (string, error) {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return "", fmt.Errorf("连接ClamAV失败: %v", err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "INSTREAM\n")
	if err != nil {
		return "", fmt.Errorf("发送INSTREAM命令失败: %v", err)
	}

	buf := make([]byte, 8192)
	for {
		n, err := data.Read(buf)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("读取数据失败: %v", err)
		}
		if n == 0 {
			break
		}

		size := uint32(n)
		err = binary.Write(conn, binary.BigEndian, size)
		if err != nil {
			return "", fmt.Errorf("发送数据大小失败: %v", err)
		}

		_, err = conn.Write(buf[:n])
		if err != nil {
			return "", fmt.Errorf("发送数据失败: %v", err)
		}
	}

	// 发送结束标记
	err = binary.Write(conn, binary.BigEndian, uint32(0))
	if err != nil {
		return "", fmt.Errorf("发送结束标记失败: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	result := scanner.Text()

	return result, nil
}

// MultiScan 并行扫描多个文件
func (c *Client) MultiScan(filePaths []string) (map[string]string, error) {
	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return nil, fmt.Errorf("连接ClamAV失败: %v", err)
	}
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "MULTISCAN %s\n", strings.Join(filePaths, " "))
	if err != nil {
		return nil, fmt.Errorf("发送MULTISCAN命令失败: %v", err)
	}

	results := make(map[string]string)
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			results[parts[0]] = strings.TrimSpace(parts[1])
		}
	}

	return results, nil
}
