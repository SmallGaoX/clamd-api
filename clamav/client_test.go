package clamav

import (
	"testing"
)

// MockScanner 是一个模拟的Scanner实现
type MockScanner struct {
	ScanFileFunc   func(filePath string) (string, error)
	GetVersionFunc func() (string, error)
}

func (m *MockScanner) ScanFile(filePath string) (string, error) {
	return m.ScanFileFunc(filePath)
}

func (m *MockScanner) GetVersion() (string, error) {
	return m.GetVersionFunc()
}

func TestScanFile(t *testing.T) {
	mockScanner := &MockScanner{
		ScanFileFunc: func(filePath string) (string, error) {
			return "OK", nil
		},
	}

	result, err := mockScanner.ScanFile("/path/to/file")
	if err != nil {
		t.Errorf("预期无错误,但得到: %v", err)
	}
	if result != "OK" {
		t.Errorf("预期结果为 'OK', 但得到: %s", result)
	}
}

func TestGetVersion(t *testing.T) {
	mockScanner := &MockScanner{
		GetVersionFunc: func() (string, error) {
			return "ClamAV 0.103.2", nil
		},
	}

	version, err := mockScanner.GetVersion()
	if err != nil {
		t.Errorf("预期无错误,但得到: %v", err)
	}
	if version != "ClamAV 0.103.2" {
		t.Errorf("预期版本为 'ClamAV 0.103.2', 但得到: %s", version)
	}
}
