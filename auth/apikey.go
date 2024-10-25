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
)

const encryptionKey = "clamav-api-secret" // 用于 XOR 加密的密钥

type APIKeyManager struct {
	apiKeys   map[string]string // 键是加密后的 API key，值是名称
	nameToKey map[string]string // 键是名称，值是加密后的 API key
	mutex     sync.RWMutex
	file      string
}

func NewAPIKeyManager(file string) (*APIKeyManager, error) {
	manager := &APIKeyManager{
		apiKeys:   make(map[string]string),
		nameToKey: make(map[string]string),
		file:      file,
	}

	err := manager.loadAPIKeys()
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *APIKeyManager) loadAPIKeys() error {
	file, err := os.Open(m.file)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(m.file, []byte{}, 0600)
		}
		return err
	}
	defer file.Close()

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

	return scanner.Err()
}

func (m *APIKeyManager) IsValidAPIKey(apiKey string) bool {
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

func (m *APIKeyManager) AddAPIKey(apiKey, name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查名称是否已存在
	if _, exists := m.nameToKey[name]; exists {
		return errors.New("API key 名称已存在")
	}

	encryptedKey := encryptAPIKey(apiKey)
	if _, exists := m.apiKeys[encryptedKey]; exists {
		return errors.New("API key 已存在")
	}

	m.apiKeys[encryptedKey] = name
	m.nameToKey[name] = encryptedKey

	// 将新的加密后的 API key 追加到文件中
	file, err := os.OpenFile(m.file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%s %s\n", encryptedKey, name)
	return err
}

func (m *APIKeyManager) RemoveAPIKey(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	encryptedKey, exists := m.nameToKey[name]
	if !exists {
		return errors.New("API key 名称不存在")
	}

	delete(m.apiKeys, encryptedKey)
	delete(m.nameToKey, name)

	err := m.rewriteAPIKeysFile()
	if err != nil {
		return err
	}

	// 重新加载 API keys
	return m.reloadAPIKeys()
}

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

func (m *APIKeyManager) reloadAPIKeys() error {
	m.apiKeys = make(map[string]string)
	m.nameToKey = make(map[string]string)
	return m.loadAPIKeys()
}

func (m *APIKeyManager) LoadAPIKeys() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.loadAPIKeys()
}

func encryptAPIKey(apiKey string) string {
	encrypted := make([]byte, len(apiKey))
	for i := 0; i < len(apiKey); i++ {
		encrypted[i] = apiKey[i] ^ encryptionKey[i%len(encryptionKey)]
	}
	return base64.StdEncoding.EncodeToString(encrypted)
}

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
	_, err := rand.Read(bytes)
	if err != nil {
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

// 添加这个辅助函数来打印调试信息
func (m *APIKeyManager) DebugPrintKeys() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	fmt.Println("Stored API Keys:")
	for encryptedKey, name := range m.apiKeys {
		fmt.Printf("Encrypted: %s, Name: %s\n", encryptedKey, name)
	}
}