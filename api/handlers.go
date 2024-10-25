package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// ScanFileHandler 处理文件扫描请求（支持单个或多个文件）
func (h *Handler) ScanFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只允许 POST 请求", http.StatusMethodNotAllowed)
		return
	}

	// 解析多部分表单数据
	err := r.ParseMultipartForm(32 << 20) // 32 MB
	if err != nil {
		http.Error(w, "解析表单数据失败", http.StatusBadRequest)
		return
	}

	results := make(map[string]string)

	// 遍历所有上传的文件
	for _, fileHeaders := range r.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				results[fileHeader.Filename] = fmt.Sprintf("打开文件失败: %v", err)
				continue
			}
			defer file.Close()

			isSafe, err := h.scanner.ScanStream(file)
			if err != nil {
				results[fileHeader.Filename] = fmt.Sprintf("扫描错误: %v", err)
			} else if isSafe {
				results[fileHeader.Filename] = "SECURE"
			} else {
				results[fileHeader.Filename] = "!!! THREAT !!!"
			}
		}
	}

	// 如果只有一个文件，返回简单的文本响应
	if len(results) == 1 {
		for _, result := range results {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(result))
			return
		}
	}

	// 如果有多个文件，返回 JSON 响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// ScanStreamHandler 处理文件列表扫描请求（支持文件路径列表和多文件上传）
func (h *Handler) ScanStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只允许 POST 请求", http.StatusMethodNotAllowed)
		return
	}

	results := make(map[string]string)

	// 检查是否是多部分表单数据
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		// 处理多文件上传
		err := r.ParseMultipartForm(32 << 20) // 32 MB
		if err != nil {
			http.Error(w, "解析表单数据失败", http.StatusBadRequest)
			return
		}

		for _, fileHeaders := range r.MultipartForm.File {
			for _, fileHeader := range fileHeaders {
				file, err := fileHeader.Open()
				if err != nil {
					results[fileHeader.Filename] = fmt.Sprintf("打开文件失败: %v", err)
					continue
				}
				defer file.Close()

				isSafe, err := h.scanner.ScanStream(file)
				if err != nil {
					results[fileHeader.Filename] = fmt.Sprintf("扫描错误: %v", err)
				} else if isSafe {
					results[fileHeader.Filename] = "SECURE"
				} else {
					results[fileHeader.Filename] = "!!! THREAT !!!"
				}
			}
		}
	} else {
		// 处理文件路径列表
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "读取请求体失败", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		filePaths := strings.Split(string(body), "\n")

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
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
