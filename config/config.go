package config

import (
"fmt"
"os"
"os/exec"
"path/filepath"
"sort"
"strings"
"time"

"github.com/BurntSushi/toml"
)

// Config は設定ファイルの構造体
type Config struct {
SSH struct {
Host           string `toml:"host"`
User           string `toml:"user"`
Port           int    `toml:"port"`
PrivateKeyPath string `toml:"private_key_path"`
Password       string `toml:"password"`
} `toml:"ssh"`
Docker struct {
Dockerfile     string `toml:"dockerfile"`
ImageName      string `toml:"image_name"`
Tag           string `toml:"tag"`
Context       string `toml:"context"`
UseCompose    bool   `toml:"use_compose"`    // Docker Compose使用フラグ
ComposeFile   string `toml:"compose_file"`   // docker-compose.ymlのパス
ServiceName   string `toml:"service_name"`   // 対象のサービス名
ComposeEnvFile string `toml:"compose_env_file"` // 環境変数ファイル
} `toml:"docker"`
Remote struct {
ContainerName string            `toml:"container_name"`
Ports         []string          `toml:"ports"`
Environment   map[string]string `toml:"environment"`
Volumes       []string          `toml:"volumes"`
} `toml:"remote"`
Deploy struct {
TriggerBranch  string `toml:"trigger_branch"`
CompressedFile string `toml:"compressed_file"`
RemoteTempDir  string `toml:"remote_temp_dir"`
} `toml:"deploy"`
Compose struct {
EnvFiles    []string `toml:"env_files"`    // 環境変数ファイル群
ExtraFiles  []string `toml:"extra_files"`  // 追加で転送が必要なファイル
TargetEnv   string   `toml:"target_env"`   // ビルド/デプロイ時の環境指定
} `toml:"compose"`
}

// LoadConfig は指定したファイルから設定を読み込む関数
func LoadConfig(path string) (Config, error) {
var conf Config
if _, err := os.Stat(path); os.IsNotExist(err) {
return conf, fmt.Errorf("設定ファイルが存在しません: %s", path)
}
if _, err := toml.DecodeFile(path, &conf); err != nil {
return conf, err
}
return conf, nil
}

// GenerateDefaultConfig はデフォルトの設定ファイルを生成する関数
func GenerateDefaultConfig(path string) error {
defaultConfig := `
[ssh]
host = "example.com"
user = "deploy"
port = 22
# private_key_path = "/path/to/private/key"
password = "your_password"  # パスワード認証を使う場合はこちら

[docker]
use_compose = false        # Docker Compose使用フラグ
compose_file = "docker-compose.yml"
service_name = "app"      # 対象のサービス名
compose_env_file = ".env" # 環境変数ファイル

# Docker Compose未使用時の設定
dockerfile = "Dockerfile"
image_name = "myapp"
context = "./"
tag = "latest"

[remote]
container_name = "myapp_container"
ports = ["80:80"]
environment = { DATABASE_URL = "your_database_url", API_KEY = "your_api_key" }
volumes = ["/data:/app/data"]

[deploy]
trigger_branch = "main"
compressed_file = "deploy.tar.gz"
remote_temp_dir = "~/tmp"

[compose]
env_files = [".env", ".env.prod"]  # 環境変数ファイル群
extra_files = [                    # 追加で転送が必要なファイル
    "nginx.conf",
    "mysql/init.sql"
]
target_env = "production"          # ビルド/デプロイ時の環境指定
`
// configディレクトリを作成
if err := os.MkdirAll("config", 0755); err != nil {
return fmt.Errorf("configディレクトリの作成に失敗: %w", err)
}

file, err := os.Create(path)
if err != nil {
return fmt.Errorf("設定ファイルの作成に失敗: %w", err)
}
defer file.Close()

_, err = file.WriteString(defaultConfig)
return err
}

// DeployHistoryEntry はデプロイ履歴のエントリ
type DeployHistoryEntry struct {
Version       string    `toml:"version"`
CommitHash    string    `toml:"commit_hash"`
CommitMessage string    `toml:"commit_message"`
Image         string    `toml:"image"`
Timestamp     time.Time `toml:"timestamp"`
TimestampTag  string    `toml:"timestamp_tag"`
ComposeInfo   struct {
ServiceName string            `toml:"service_name,omitempty"`
EnvFiles    []string         `toml:"env_files,omitempty"`
ExtraFiles  []string         `toml:"extra_files,omitempty"`
Config      map[string]any   `toml:"config,omitempty"`
} `toml:"compose_info,omitempty"`
}

// History はバージョン識別子をキーとしたデプロイ履歴のマップ
type History map[string]DeployHistoryEntry

// LoadHistory は指定したファイルから履歴を読み込む関数
func LoadHistory(path string) (History, error) {
var history History
if _, err := os.Stat(path); os.IsNotExist(err) {
// 履歴ファイルが無ければ空のマップを返す
return make(History), nil
}
if _, err := toml.DecodeFile(path, &history); err != nil {
return nil, err
}
return history, nil
}

// RecordDeployHistory は新たなデプロイ履歴エントリを記録する関数
func RecordDeployHistory(conf Config) error {
historyPath := "config/history.toml"
history, err := LoadHistory(historyPath)
if err != nil {
return err
}
// タイムスタンプベースのバージョン識別子を生成
now := time.Now()
version := fmt.Sprintf("%d", now.Unix())

// Git commit hash とメッセージを取得
commitHashBytes, err := execCommand("git", "rev-parse", "--short", "HEAD")
commitHash := "unknown"
if err == nil {
commitHash = string(commitHashBytes)
}

// コミットメッセージを取得
commitMsgBytes, err := execCommand("git", "log", "-1", "--pretty=%B")
commitMsg := "unknown"
if err == nil {
commitMsg = string(commitMsgBytes)
}

entry := DeployHistoryEntry{
Version:       version,
CommitHash:    commitHash,
CommitMessage: commitMsg,
Image:         fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag),
Timestamp:     now,
TimestampTag:  conf.Docker.Tag,
}

// Docker Compose使用時は追加情報を記録
if conf.Docker.UseCompose {
entry.ComposeInfo.ServiceName = conf.Docker.ServiceName
entry.ComposeInfo.EnvFiles = conf.Compose.EnvFiles
entry.ComposeInfo.ExtraFiles = conf.Compose.ExtraFiles
entry.ComposeInfo.Config = map[string]any{
"target_env": conf.Compose.TargetEnv,
}
}

history[version] = entry

// 履歴を TOML ファイルに書き出す
file, err := os.Create(historyPath)
if err != nil {
return err
}
defer file.Close()

enc := toml.NewEncoder(file)
return enc.Encode(history)
}

// ShowDeployHistory は履歴を表示する関数
func ShowDeployHistory() ([]string, error) {
wd, err := os.Getwd()
if err != nil {
return nil, err
}
historyPath := filepath.Join(wd, "config/history.toml")
history, err := LoadHistory(historyPath)
if err != nil {
return nil, err
}

// バージョンのリストを作成
versions := make([]string, 0, len(history))
for version := range history {
versions = append(versions, version)
}

// バージョンを時系列順にソート（新しい順）
sort.Sort(sort.Reverse(sort.StringSlice(versions)))

fmt.Println("\n============= Deploy History =============\n")

// 各バージョンの情報を表示
for _, version := range versions {
entry := history[version]
fmt.Printf("[Version] %s\n", version)
fmt.Println("┌─────────────┬──────────────────────────────────┐")
fmt.Printf("│ Commit Hash │ %-30s │\n", entry.CommitHash)
fmt.Printf("│ Message     │ %-30s │\n", truncateString(entry.CommitMessage, 30))
fmt.Printf("│ Image       │ %-30s │\n", entry.Image)
fmt.Printf("│ Time        │ %-30s │\n", entry.Timestamp.Format("2006-01-02 15:04:05 MST"))

// Docker Compose情報がある場合は表示
if entry.ComposeInfo.ServiceName != "" {
fmt.Printf("│ Service     │ %-30s │\n", entry.ComposeInfo.ServiceName)
fmt.Printf("│ Env Files   │ %-30s │\n", truncateString(strings.Join(entry.ComposeInfo.EnvFiles, ","), 30))
if targetEnv, ok := entry.ComposeInfo.Config["target_env"]; ok {
fmt.Printf("│ Target Env  │ %-30s │\n", targetEnv)
}
}

fmt.Println("└─────────────┴──────────────────────────────────┘")
fmt.Println()
}

fmt.Println("=========================================")
return versions, nil
}

// truncateString は文字列を指定した表示幅に切り詰める関数
func truncateString(str string, width int) string {
str = strings.TrimSpace(str)
currentWidth := 0
for i, r := range str {
w := 1
if r > 0x7F {
w = 2 // 全角文字は幅2としてカウント
}
if currentWidth+w > width {
if i > 0 {
return str[:i] + "..."
}
return str[:1] + "..." // 最低1文字は表示
}
currentWidth += w
}
return str
}

// execCommand はコマンド実行して出力を返すヘルパー関数
func execCommand(name string, args ...string) ([]byte, error) {
cmd := exec.Command(name, args...)
// 日本語対応のための環境変数設定
cmd.Env = append(os.Environ(), "LANG=ja_JP.UTF-8")
out, err := cmd.Output()
if err != nil {
return out, err
}
// 改行を削除
return []byte(strings.TrimSpace(string(out))), nil
}

// GenerateDefaultDockerfile はデフォルトのDockerfileを生成する関数
func GenerateDefaultDockerfile(path string) error {
defaultDockerfile := `# 使用するベースイメージ
FROM alpine:latest

# コンテナ実行時に実行されるコマンド
CMD echo "Hello, World!"
`
file, err := os.Create(path)
if err != nil {
return fmt.Errorf("Dockerfileの作成に失敗: %w", err)
}
defer file.Close()

_, err = file.WriteString(defaultDockerfile)
return err
}
