package config

import (
"os"
"path/filepath"
"testing"
"time"

"github.com/BurntSushi/toml"
)

func TestLoadConfig(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// 異なるケース用の設定ファイルを作成
	tests := []struct {
		name     string
		content  string
		wantErr  bool
		validate func(*testing.T, Config)
	}{
		{
			name: "通常のDockerfile設定",
			content: `
[docker]
use_compose = false
dockerfile = "Dockerfile"
image_name = "test-app"
tag = "latest"
context = "./"

[remote]
container_name = "test-container"
ports = ["80:80"]
`,
			wantErr: false,
			validate: func(t *testing.T, conf Config) {
				if conf.Docker.UseCompose {
					t.Error("UseComposeがtrueになっています")
				}
				if conf.Docker.ImageName != "test-app" {
					t.Errorf("ImageName: want test-app, got %s", conf.Docker.ImageName)
				}
			},
		},
		{
			name: "Docker Compose設定",
			content: `
[docker]
use_compose = true
compose_file = "docker-compose.yml"
service_name = "web"
compose_env_file = ".env"

[compose]
env_files = [".env", ".env.prod"]
extra_files = ["nginx.conf"]
target_env = "production"
`,
			wantErr: false,
			validate: func(t *testing.T, conf Config) {
				if !conf.Docker.UseCompose {
					t.Error("UseComposeがfalseになっています")
				}
				if conf.Docker.ServiceName != "web" {
					t.Errorf("ServiceName: want web, got %s", conf.Docker.ServiceName)
				}
				if len(conf.Compose.EnvFiles) != 2 {
					t.Errorf("EnvFiles: want 2, got %d", len(conf.Compose.EnvFiles))
				}
			},
		},
		{
			name: "不正なTOML",
			content: `
[docker
use_compose = true
`,
			wantErr: true,
			validate: func(t *testing.T, conf Config) {
				// エラーケースなのでvalidation不要
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テストファイルを作成
			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// 設定を読み込み
			conf, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				tt.validate(t, conf)
			}
		})
	}
}

func TestGenerateDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config/config.toml")

	// デフォルト設定を生成
	err := GenerateDefaultConfig(configPath)
	if err != nil {
		t.Fatalf("GenerateDefaultConfig() error = %v", err)
	}

	// 生成された設定を読み込んでテスト
	conf, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// デフォルト値の検証
	if conf.Docker.ImageName != "myapp" {
		t.Errorf("デフォルトのImageName: want myapp, got %s", conf.Docker.ImageName)
	}

	if conf.Docker.UseCompose {
		t.Error("デフォルトのUseComposeがtrueになっています")
	}
}

func TestShowDeployHistory(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "config/history.toml")

	// テスト用のディレクトリ構造を作成
	if err := os.MkdirAll(filepath.Dir(historyPath), 0755); err != nil {
		t.Fatal(err)
	}

	// テスト用の履歴データを作成
	history := History{
		"1234567890": {
			Version:       "1234567890",
			CommitHash:    "abc123",
			CommitMessage: "テストコミット",
			Image:         "test-app:latest",
			Timestamp:     time.Now(),
			TimestampTag:  "20240322",
			ComposeInfo: struct {
				ServiceName string          `toml:"service_name,omitempty"`
				EnvFiles    []string       `toml:"env_files,omitempty"`
				ExtraFiles  []string       `toml:"extra_files,omitempty"`
				Config      map[string]any `toml:"config,omitempty"`
			}{
				ServiceName: "web",
				EnvFiles:    []string{".env"},
				Config: map[string]any{
					"target_env": "production",
				},
			},
		},
	}

	// 履歴ファイルを作成
	file, err := os.Create(historyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(history); err != nil {
		t.Fatal(err)
	}

	// カレントディレクトリを一時的に変更
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalWd)

	// 履歴表示をテスト
	versions, err := ShowDeployHistory()
	if err != nil {
		t.Fatalf("ShowDeployHistory() error = %v", err)
	}

	if len(versions) != 1 {
		t.Errorf("バージョン数: want 1, got %d", len(versions))
	}

	if versions[0] != "1234567890" {
		t.Errorf("バージョン: want 1234567890, got %s", versions[0])
	}
}

func TestRecordDeployHistory(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	// テスト用の設定
	conf := Config{
		Docker: struct {
			Dockerfile     string `toml:"dockerfile"`
			ImageName      string `toml:"image_name"`
			Tag           string `toml:"tag"`
			Context       string `toml:"context"`
			UseCompose    bool   `toml:"use_compose"`
			ComposeFile   string `toml:"compose_file"`
			ServiceName   string `toml:"service_name"`
			ComposeEnvFile string `toml:"compose_env_file"`
		}{
			ImageName:   "test-app",
			Tag:        "latest",
			UseCompose: true,
			ServiceName: "web",
		},
		Compose: struct {
			EnvFiles    []string `toml:"env_files"`
			ExtraFiles  []string `toml:"extra_files"`
			TargetEnv   string   `toml:"target_env"`
		}{
			EnvFiles:  []string{".env"},
			TargetEnv: "production",
		},
	}

	// デプロイ履歴を記録
	if err := RecordDeployHistory(conf); err != nil {
		t.Fatalf("RecordDeployHistory() error = %v", err)
	}

	// 記録された履歴を読み込んで検証
	history, err := LoadHistory("config/history.toml")
	if err != nil {
		t.Fatalf("LoadHistory() error = %v", err)
	}

	if len(history) != 1 {
		t.Errorf("履歴エントリ数: want 1, got %d", len(history))
	}

	// 最新のエントリを取得
	var entry DeployHistoryEntry
	for _, e := range history {
		entry = e
		break
	}

	if entry.Image != "test-app:latest" {
		t.Errorf("イメージ名: want test-app:latest, got %s", entry.Image)
	}

	if entry.ComposeInfo.ServiceName != "web" {
		t.Errorf("サービス名: want web, got %s", entry.ComposeInfo.ServiceName)
	}
}
