package auth

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const encryptionKey = "clamav-api-secret" // 用于 XOR 加密的密钥

// APIKeyManager 管理 API keys
type APIKeyManager struct {
	apiKeys     map[string]string // 键是加密后的 API key，值是名称
	nameToKey   map[string]string // 键是名称，值是加密后的 API key
	mutex       sync.RWMutex
	file        string
	lastModTime time.Time
}

// NewAPIKeyManager 创建一个新的 APIKeyManager
func NewAPIKeyManager(file string) (*APIKeyManager, error) {
	manager := &APIKeyManager{
		apiKeys:   make(map[string]string),
		nameToKey: make(map[string]string),
		file:      file,
	}

	// 检查文件是否存在，如果不存在则创建
	if _, err := os.Stat(file); os.IsNotExist(err) {
		_, err := os.Create(file)
		if err != nil {
			return nil, fmt.Errorf("创建 API keys 文件失败: %v", err)
		}
	}

	err := manager.loadAPIKeys()
	if err != nil {
		return nil, err
	}

	return manager, nil
}

// loadAPIKeys 从文件加载 API keys
func (m *APIKeyManager) loadAPIKeys() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	fileInfo, err := os.Stat(m.file)
	if err != nil {
		return err
	}

	if fileInfo.Size() == 0 {
		// 文件为空，不需要加载任何内容
		m.lastModTime = fileInfo.ModTime()
		return nil
	}

	if fileInfo.ModTime() == m.lastModTime {
		return nil // 文件未被修改，无需重新加载
	}

	file, err := os.Open(m.file)
	if err != nil {
		return err
	}
	defer file.Close()

	m.apiKeys = make(map[string]string)
	m.nameToKey = make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			encryptedKey := parts[0]
			name := parts[1]
			m.apiKeys[encryptedKey] = name
			m.nameToKey[name] = encryptedKey
		}
	}

	m.lastModTime = fileInfo.ModTime()

	return scanner.Err()
}

// IsValidAPIKey 检查 API key 是否有效
func (m *APIKeyManager) IsValidAPIKey(apiKey string) bool {
	if err := m.loadAPIKeys(); err != nil {
		fmt.Printf("重新加载 API keys 时出错: %v\n", err)
		return false
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()
	encryptedKey := encryptAPIKey(apiKey)
	_, exists := m.apiKeys[encryptedKey]
	return exists
}

// GetAPIKeyName 返回给定 API key 的名称
func (m *APIKeyManager) GetAPIKeyName(apiKey string) (string, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	encryptedKey := encryptAPIKey(apiKey)
	name, exists := m.apiKeys[encryptedKey]
	return name, exists
}

// AddAPIKey 添加新的 API key
func (m *APIKeyManager) AddAPIKey(apiKey, name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.nameToKey[name]; exists {
		return errors.New("API key 名称已存在")
	}

	encryptedKey := encryptAPIKey(apiKey)
	if _, exists := m.apiKeys[encryptedKey]; exists {
		return errors.New("API key 已存在")
	}

	m.apiKeys[encryptedKey] = name
	m.nameToKey[name] = encryptedKey

	file, err := os.OpenFile(m.file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%s %s\n", encryptedKey, name)
	return err
}

// RemoveAPIKey 通过名称删除 API key
func (m *APIKeyManager) RemoveAPIKey(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查是否需要重新加载
	if err := m.checkAndReload(); err != nil {
		return fmt.Errorf("重新加载 API keys 失败: %v", err)
	}

	encryptedKey, exists := m.nameToKey[name]
	if !exists {
		return fmt.Errorf("未找到名称为 '%s' 的 API key", name)
	}

	delete(m.apiKeys, encryptedKey)
	delete(m.nameToKey, name)

	// 立即保存更改
	return m.saveAPIKeys()
}

// 添加一个新的方法来检查是否需要重新加载
func (m *APIKeyManager) checkAndReload() error {
	fileInfo, err := os.Stat(m.file)
	if err != nil {
		return err
	}

	if fileInfo.ModTime() != m.lastModTime {
		return m.loadAPIKeys()
	}

	return nil
}

// rewriteAPIKeysFile 重写 API keys 文件
func (m *APIKeyManager) rewriteAPIKeysFile() error {
	file, err := os.Create(m.file)
	if err != nil {
		return err
	}
	defer file.Close()

	for encryptedKey, name := range m.apiKeys {
		_, err := fmt.Fprintf(file, "%s %s\n", encryptedKey, name)
		if err != nil {
			return err
		}
	}

	return nil
}

// reloadAPIKeys 重新加载 API keys
func (m *APIKeyManager) reloadAPIKeys() error {
	m.apiKeys = make(map[string]string)
	m.nameToKey = make(map[string]string)
	return m.loadAPIKeys()
}

// LoadAPIKeys 加载 API keys
func (m *APIKeyManager) LoadAPIKeys() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.loadAPIKeys()
}

// encryptAPIKey 加密 API key
func encryptAPIKey(apiKey string) string {
	encrypted := make([]byte, len(apiKey))
	for i := 0; i < len(apiKey); i++ {
		encrypted[i] = apiKey[i] ^ encryptionKey[i%len(encryptionKey)]
	}
	return base64.StdEncoding.EncodeToString(encrypted)
}

// decryptAPIKey 解密 API key
func decryptAPIKey(encryptedKey string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", err
	}
	decrypted := make([]byte, len(decoded))
	for i := 0; i < len(decoded); i++ {
		decrypted[i] = decoded[i] ^ encryptionKey[i%len(encryptionKey)]
	}
	return string(decrypted), nil
}

// GetAllAPIKeys 返回所有的 API key 及其备注
func (m *APIKeyManager) GetAllAPIKeys() (map[string]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]string)
	for encryptedKey, name := range m.apiKeys {
		decryptedKey, err := decryptAPIKey(encryptedKey)
		if err != nil {
			return nil, fmt.Errorf("解密 API key 失败: %v", err)
		}
		result[decryptedKey] = name
	}

	return result, nil
}

// GenerateAPIKey 生成一个随机的 API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 256 位
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GetAllObfuscatedAPIKeys 返回所有的混淆后的 API key 及其备注
func (m *APIKeyManager) GetAllObfuscatedAPIKeys() map[string]string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]string)
	for encryptedKey, name := range m.apiKeys {
		obfuscatedKey := obfuscateAPIKey(encryptedKey)
		result[name] = obfuscatedKey
	}

	return result
}

// obfuscateAPIKey 混淆 API key,只显示前4个和后4个字符
func obfuscateAPIKey(encryptedKey string) string {
	if len(encryptedKey) <= 8 {
		return encryptedKey
	}
	return encryptedKey[:4] + "..." + encryptedKey[len(encryptedKey)-4:]
}

// DebugPrintKeys 打印存储的 API keys（用于调试）
func (m *APIKeyManager) DebugPrintKeys() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	fmt.Println("存储的 API Keys:")
	for encryptedKey, name := range m.apiKeys {
		fmt.Printf("加密的: %s, 名称: %s\n", encryptedKey, name)
	}
}

// GetFilePath 返回 API key 文件的路径
func (m *APIKeyManager) GetFilePath() string {
	return m.file
}

// saveAPIKeys 保存 API keys
func (m *APIKeyManager) saveAPIKeys() error {
	file, err := os.Create(m.file)
	if err != nil {
		return err
	}
	defer file.Close()

	for encryptedKey, name := range m.apiKeys {
		_, err := fmt.Fprintf(file, "%s %s\n", encryptedKey, name)
		if err != nil {
			return err
		}
	}

	return nil
}
