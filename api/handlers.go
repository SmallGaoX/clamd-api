package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"clamd-api/auth"
	"clamd-api/clamav"
	"clamd-api/config"
)

// Handler 结构体包含所有API处理程序
type Handler struct {
	scanner       clamav.Scanner
	config        *config.Config
	apiKeyManager *auth.APIKeyManager
}

// NewHandler 创建一个新的Handler实例
func NewHandler(scanner clamav.Scanner, cfg *config.Config, apiKeyManager *auth.APIKeyManager) *Handler {
	return &Handler{
		scanner:       scanner,
		config:        cfg,
		apiKeyManager: apiKeyManager,
	}
}

// ScanHandler 处理文件扫描请求
func (h *Handler) ScanHandler(w http.ResponseWriter, r *http.Request) {
	// 获取上传的文件
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "获取上传文件失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 扫描文件
	isSafe, err := h.scanner.ScanStream(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("扫描文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回扫描结果
	w.Header().Set("Content-Type", "text/plain")
	if isSafe {
		w.Write([]byte("ALL GOOD"))
	} else {
		w.Write([]byte("!!! VIRUS FOUND !!!"))
	}
}

// saveUploadedFile 保存上传的文件到指定目录
func saveUploadedFile(fileHeader *multipart.FileHeader, dir string) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("打开上传文件失败: %v", err)
	}
	defer src.Close()

	dstPath := filepath.Join(dir, fileHeader.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return "", fmt.Errorf("复制文件内容失败: %v", err)
	}

	return dstPath, nil
}

// VersionHandler 处理获取ClamAV版本的请求
func (h *Handler) VersionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "只支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	version, err := h.scanner.GetVersion()
	if err != nil {
		http.Error(w, "获取版本失败", http.StatusInternalServerError)
		return
	}

	response := map[string]string{"version": version}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PingHandler 处理Ping请求
func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "只支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	err := h.scanner.Ping()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ping失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("PONG"))
}

// ReloadHandler 处理重新加载病毒数据库请求
func (h *Handler) ReloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	err := h.scanner.Reload()
	if err != nil {
		http.Error(w, fmt.Sprintf("重新加载失败: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("病毒数据库已重新加载"))
}

// ScanFileListHandler 处理文件列表扫描请求
func (h *Handler) ScanFileListHandler(w http.ResponseWriter, r *http.Request) {
	// 确保是 POST 请求
	if r.Method != http.MethodPost {
		http.Error(w, "只允许 POST 请求", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求体失败", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 将请求体分割成文件路径列表
	filePaths := strings.Split(string(body), "\n")

	results := make(map[string]string)

	for _, filePath := range filePaths {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}

		isSafe, err := h.scanner.ScanFile(filePath)
		if err != nil {
			results[filePath] = fmt.Sprintf("扫描错误: %v", err)
		} else if isSafe {
			results[filePath] = "ALL GOOD"
		} else {
			results[filePath] = "!!! VIRUS FOUND !!!"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
