package api

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

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
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析multipart表单
	err := r.ParseMultipartForm(10 << 20) // 限制上传文件大小为10MB
	if err != nil {
		http.Error(w, "解析表单失败", http.StatusBadRequest)
		return
	}

	// 获取上传的文件
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "获取上传文件失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 保存文件到临时目录
	tempFile, err := saveUploadedFile(header, h.config.TempDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("保存文件失败: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile) // 扫描完成后删除临时文件

	// 扫描文件
	result, err := h.scanner.ScanFile(tempFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("扫描文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 返回扫描结果
	response := map[string]string{"result": result}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
