package clamav

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Scanner 接口定义了防病毒扫描器的行为
type Scanner interface {
	ScanFile(filePath string) (string, error)
	GetVersion() (string, error)
	Ping() error
	Reload() error
	Shutdown() error
	ScanStream(reader io.Reader) (string, error)
}

// Client 结构体表示ClamAV客户端
type Client struct {
	address string
}

// NewClient 创建一个新的ClamAV客户端
func NewClient(address string) Scanner {
	return &Client{address: address}
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

// ScanFile 扫描单个文件
func (c *Client) ScanFile(filePath string) (string, error) {
	conn, err := net.DialTimeout("tcp", c.address, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("连接ClamAV失败: %v", err)
	}
	defer conn.Close()

	// 发送SCAN命令
	_, err = fmt.Fprintf(conn, "SCAN %s\n", filePath)
	if err != nil {
		return "", fmt.Errorf("发送SCAN命令失败: %v", err)
	}

	// 读取扫描结果
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("读取扫描结果失败: %v", err)
	}

	return strings.TrimSpace(response), nil
}

// ScanStream 扫描文件流
func (c *Client) ScanStream(reader io.Reader) (string, error) {
	conn, err := net.DialTimeout("tcp", c.address, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("连接ClamAV失败: %v", err)
	}
	defer conn.Close()

	// 设置超时
	err = conn.SetDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		return "", fmt.Errorf("设置超时失败: %v", err)
	}

	// 发送INSTREAM命令
	_, err = conn.Write([]byte("zINSTREAM\x00"))
	if err != nil {
		return "", fmt.Errorf("发送INSTREAM命令失败: %v", err)
	}

	// 发送文件内容
	buf := make([]byte, 2048)
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("读取文件流失败: %v", err)
		}
		if n == 0 {
			break
		}

		err = binary.Write(conn, binary.BigEndian, uint32(n))
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

	// 读取扫描结果
	response, err := bufio.NewReader(conn).ReadString('\x00')
	if err != nil {
		return "", fmt.Errorf("读取扫描结果失败: %v", err)
	}

	return strings.TrimSpace(strings.TrimSuffix(response, "\x00")), nil
}
