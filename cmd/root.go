package cmd

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"clamd-api/api"
	"clamd-api/auth"
	"clamd-api/clamav"
	"clamd-api/config"
)

var (
	cfgFile       string
	apiKeyManager *auth.APIKeyManager
)

var rootCmd = &cobra.Command{
	Use:   "clamd-api",
	Short: "ClamAV REST API服务",
	Long:  `ClamAV REST API服务提供了一个HTTP接口来扫描文件和获取ClamAV版本信息。`,
	Run:   runServer,
}

var apiKeyCmd = &cobra.Command{
	Use:   "apikey",
	Short: "管理 API keys",
	Long:  `添加、删除、列出 API keys。`,
}

var addAPIKeyCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "添加新的 API key",
	Long:  `添加新的 API key 到系统中,并为其添加一个名称（备注）。API key 将被自动生成。`,
	Args:  cobra.ExactArgs(1),
	Run:   addAPIKey,
}

var removeAPIKeyCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "删除指定名称的 API key",
	Long:  `通过名称从系统中删除指定的 API key。`,
	Args:  cobra.ExactArgs(1),
	Run:   removeAPIKey,
}

var listAPIKeysCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有的 API key",
	Long:  `显示系统中所有的 API key 及其对应的名称。`,
	Run:   listAPIKeys,
}

func init() {
	cobra.OnInitialize(initConfig, initAPIKeyManager)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认为 ./config.yaml)")
	rootCmd.PersistentFlags().String("clamav_address", "localhost:3310", "ClamAV服务器地址")
	rootCmd.PersistentFlags().String("temp_dir", "/tmp", "临时文件目录")
	rootCmd.PersistentFlags().String("port", "8080", "API服务器端口")
	rootCmd.PersistentFlags().String("api_key_file", "./api_keys.txt", "API key 文件路径")

	viper.BindPFlag("clamav_address", rootCmd.PersistentFlags().Lookup("clamav_address"))
	viper.BindPFlag("temp_dir", rootCmd.PersistentFlags().Lookup("temp_dir"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("api_key_file", rootCmd.PersistentFlags().Lookup("api_key_file"))

	apiKeyCmd.AddCommand(addAPIKeyCmd)
	apiKeyCmd.AddCommand(removeAPIKeyCmd)
	apiKeyCmd.AddCommand(listAPIKeysCmd)

	rootCmd.AddCommand(apiKeyCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("使用配置文件:", viper.ConfigFileUsed())
	}
}

func initAPIKeyManager() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	apiKeyManager, err = auth.NewAPIKeyManager(cfg.APIKeyFile)
	if err != nil {
		log.Fatalf("创建 API key 管理器失败: %v", err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	scanner := clamav.NewClient(cfg.ClamAVAddress)

	handler := api.NewHandler(scanner, cfg, apiKeyManager)

	// 设置路由
	http.HandleFunc("/scan", api.LoggingMiddleware(api.AuthMiddleware(handler.ScanHandler, apiKeyManager)))
	http.HandleFunc("/version", api.LoggingMiddleware(api.AuthMiddleware(handler.VersionHandler, apiKeyManager)))
	http.HandleFunc("/ping", api.LoggingMiddleware(api.AuthMiddleware(handler.PingHandler, apiKeyManager)))
	http.HandleFunc("/reload", api.LoggingMiddleware(api.AuthMiddleware(handler.ReloadHandler, apiKeyManager)))
	http.HandleFunc("/stats", api.LoggingMiddleware(api.AuthMiddleware(handler.StatsHandler, apiKeyManager)))
	http.HandleFunc("/stream-scan", api.LoggingMiddleware(api.AuthMiddleware(handler.StreamScanHandler, apiKeyManager)))
	http.HandleFunc("/multi-scan", api.LoggingMiddleware(api.AuthMiddleware(handler.MultiScanHandler, apiKeyManager)))

	// 启动服务器
	log.Printf("启动服务器,监听端口 %s...", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, nil))
}

func addAPIKey(cmd *cobra.Command, args []string) {
	name := args[0]

	apiKey, err := auth.GenerateAPIKey()
	if err != nil {
		log.Fatalf("生成 API key 失败: %v", err)
	}

	err = apiKeyManager.AddAPIKey(apiKey, name)
	if err != nil {
		log.Fatalf("添加 API key 失败: %v", err)
	}

	fmt.Printf("成功添加 API key:\n名称: %s\nAPI Key: %s\n\n", name, apiKey)
	fmt.Println("请保存此 API key，因为它不会再次显示。")
}

func removeAPIKey(cmd *cobra.Command, args []string) {
	name := args[0]

	err := apiKeyManager.RemoveAPIKey(name)
	if err != nil {
		log.Fatalf("删除 API key 失败: %v", err)
	}

	fmt.Printf("成功删除名称为 '%s' 的 API key\n", name)
}

func listAPIKeys(cmd *cobra.Command, args []string) {
	apiKeys := apiKeyManager.GetAllObfuscatedAPIKeys()

	if len(apiKeys) == 0 {
		fmt.Println("当前没有 API key")
		return
	}

	fmt.Println("API Keys:")
	fmt.Println("--------------------------------------------------")

	var sortedNames []string
	for name := range apiKeys {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		fmt.Printf("名称: %s\nAPI Key: %s\n\n", name, apiKeys[name])
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("执行命令失败: %v", err)
	}
}