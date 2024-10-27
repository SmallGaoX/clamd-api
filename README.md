# ClamAV REST API 服务

这是一个基于 Go 语言实现的 ClamAV REST API 服务，提供了一个 HTTP 接口来扫描文件和获取 ClamAV 版本信息。

## 功能特性

1. 文件扫描：支持单文件和多文件上传扫描
2. 文件流扫描：支持文件流和文件路径列表扫描
3. ClamAV 版本查询
4. ClamAV 服务器 Ping 测试
5. ClamAV 病毒数据库重新加载
6. API Key 认证和管理
7. 日志记录
8. 版本信息查看

## 项目结构

```
.
├── api/
│   ├── handlers.go    # API 请求处理函数
│   └── middleware.go  # 中间件（日志记录和认证）
├── auth/
│   └── apikey.go      # API Key 管理
├── clamav/
│   └── client.go      # ClamAV 客户端
├── cmd/
│   └── root.go        # 命令行接口
├── config/
│   └── config.go      # 配置加载
├── version/
│   └── version.go     # 版本信息
├── main.go            # 程序入口
└── README.md          # 项目文档
```

## 实现细节

1. **API 处理**：使用标准库 `net/http` 实现 HTTP 服务器。
2. **ClamAV 客户端**：通过 TCP 连接与 ClamAV 守护进程通信。
3. **配置管理**：使用 `viper` 库加载和管理配置。
4. **命令行接口**：使用 `cobra` 库实现命令行功能。
5. **API Key 管理**：实现了基于文件的 API Key 存储和验证机制，支持加密存储。
6. **中间件**：实现了日志记录和 API Key 认证中间件。
7. **版本信息**：在构建时注入版本信息，可通过命令行查看。

## 使用方法

### 安装

1. 克隆仓库：
   ```
   git clone https://github.com/your-username/clamd-api.git
   ```

2. 进入项目目录：
   ```
   cd clamd-api
   ```

3. 构建项目：
   ```
   go build
   ```

### 配置

创建 `config.yaml` 文件，包含以下配置项：

```yaml
clamav_address: "localhost:3310"
temp_dir: "/tmp"
port: "8080"
api_key_file: "api_keys.txt"
log_file: "clamd-api.log"
```

### 运行

启动服务器：

```
./clamd-api
```

### API 使用

1. 扫描文件：
   ```
   POST /scan
   Header: X-API-Key: <your-api-key>
   Body: multipart/form-data
   ```

2. 扫描文件流或文件路径列表：
   ```
   POST /stream
   Header: X-API-Key: <your-api-key>
   Body: multipart/form-data 或 文本文件路径列表
   ```

3. 获取 ClamAV 版本：
   ```
   GET /version
   Header: X-API-Key: <your-api-key>
   ```

4. Ping ClamAV 服务器：
   ```
   GET /ping
   Header: X-API-Key: <your-api-key>
   ```

5. 重新加载病毒数据库：
   ```
   POST /reload
   Header: X-API-Key: <your-api-key>
   ```

### API 响应格式

扫描结果将以 JSON 数组的形式返回，每个元素包含以下字段：

```json
[
    {
        "fileName": "example.txt",
        "isSafe": true,
        "threat": ""
    },
    {
        "fileName": "virus.exe",
        "isSafe": false,
        "threat": "Win.Trojan.Example-1"
    }
]
```

### API Key 管理

1. 添加 API Key：
   ```
   ./clamd-api apikey add <name>
   ```

2. 删除 API Key：
   ```
   ./clamd-api apikey del <name>
   ```

3. 列出所有 API Key：
   ```
   ./clamd-api apikey list
   ```

### 版本信息

查看应用程序版本信息：

```
./clamd-api version
```

## 日志记录

应用程序会将日志记录到文件中。默认的日志文件名为 `clamd-api.log`，位于程序运行的当前目录。
您可以通过配置文件或命令行参数修改日志文件的位置。

## 注意事项

- 确保 ClamAV 守护进程正在运行并可访问。
- API Key 文件默认位于程序所在目录，可通过配置文件或命令行参数修改。
- 妥善保管 API Key，不要泄露给未授权的用户。
- 定期更新 ClamAV 病毒数据库以确保最新的病毒检测能力。

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目。

## 许可证

[MIT License](LICENSE)