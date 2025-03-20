package config

import (
    "fmt"
    "os"
    "os/exec"
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
        Dockerfile string `toml:"dockerfile"`
        ImageName  string `toml:"image_name"`
        Tag        string `toml:"tag"`
        Context    string `toml:"context"`
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
dockerfile = "Dockerfile"
image_name = "myapp"
tag = "latest"
context = "./"

[remote]
container_name = "myapp_container"
ports = ["80:80", "443:443"]
environment = { DATABASE_URL = "your_database_url", API_KEY = "your_api_key" }
volumes = ["/data:/app/data"]

[deploy]
trigger_branch = "main"
compressed_file = "deploy.tar.gz"
remote_temp_dir = "/tmp"
`
    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = file.WriteString(defaultConfig)
    return err
}

// DeployHistoryEntry はデプロイ履歴のエントリ
type DeployHistoryEntry struct {
    Version    string    `toml:"version"`
    CommitHash string    `toml:"commit_hash"`
    Image      string    `toml:"image"`
    Timestamp  time.Time `toml:"timestamp"`
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
    // バージョン識別子は現在の Unix タイムスタンプを利用
    version := fmt.Sprintf("%d", time.Now().Unix())
    // Git commit hash を取得（失敗したら unknown）
    commitHashBytes, err := execCommand("git", "rev-parse", "--short", "HEAD")
    commitHash := "unknown"
    if err == nil {
        commitHash = string(commitHashBytes)
    }
    entry := DeployHistoryEntry{
        Version:    version,
        CommitHash: commitHash,
        Image:      fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag),
        Timestamp:  time.Now(),
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
func ShowDeployHistory() error {
    historyPath := "config/history.toml"
    history, err := LoadHistory(historyPath)
    if err != nil {
        return err
    }
    for version, entry := range history {
        fmt.Printf("Version: %s, Commit: %s, Image: %s, Time: %s\n", version, entry.CommitHash, entry.Image, entry.Timestamp.Format(time.RFC3339))
    }
    return nil
}

// execCommand はコマンド実行して出力を返すヘルパー関数
func execCommand(name string, args ...string) ([]byte, error) {
    cmd := exec.Command(name, args...)
    return cmd.Output()
}
