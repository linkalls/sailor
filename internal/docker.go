package internal

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/linkalls/sailor/config"
	"golang.org/x/crypto/ssh"
)

// ExecuteComposeCommand はDocker Composeコマンドを実行する関数
func ExecuteComposeCommand(conf *config.Config, args ...string) error {
	baseArgs := []string{
		"-f", conf.Docker.ComposeFile,
	}

	if conf.Docker.ComposeEnvFile != "" {
		baseArgs = append(baseArgs, "--env-file", conf.Docker.ComposeEnvFile)
	}

	if conf.Compose.TargetEnv != "" {
		baseArgs = append(baseArgs, "--profile", conf.Compose.TargetEnv)
	}

	args = append(baseArgs, args...)
	cmd := exec.Command("docker-compose", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// TransferComposeFiles は docker-compose.yml と関連ファイルを転送する関数
func TransferComposeFiles(conf config.Config) error {
	// まず docker-compose.yml を転送
	if err := TransferFile(conf, conf.Docker.ComposeFile, conf.Deploy.RemoteTempDir+"/"+conf.Docker.ComposeFile); err != nil {
		return fmt.Errorf("docker-compose.ymlの転送に失敗: %w", err)
	}

	// 環境変数ファイルの転送
	for _, envFile := range conf.Compose.EnvFiles {
		remotePath := conf.Deploy.RemoteTempDir + "/" + envFile
		if err := TransferFile(conf, envFile, remotePath); err != nil {
			return fmt.Errorf("環境変数ファイル %s の転送に失敗: %w", envFile, err)
		}
	}

	// 追加ファイルの転送
	for _, extraFile := range conf.Compose.ExtraFiles {
		remotePath := conf.Deploy.RemoteTempDir + "/" + extraFile
		if err := TransferFile(conf, extraFile, remotePath); err != nil {
			return fmt.Errorf("追加ファイル %s の転送に失敗: %w", extraFile, err)
		}
	}

	return nil
}

// BuildDockerImage は Docker イメージをビルドする関数
func BuildDockerImage(conf *config.Config) error {
	fmt.Println("Dockerイメージをビルド中...")

	// タイムスタンプベースのタグを生成
	timestamp := time.Now().Format("20060102150405")
	conf.Docker.Tag = timestamp

	if conf.Docker.UseCompose {
		// Docker Composeでビルド
		if err := ExecuteComposeCommand(conf, "build", conf.Docker.ServiceName); err != nil {
			return fmt.Errorf("Docker Composeのビルドに失敗: %w", err)
		}
	} else {
		// 従来のDockerfileでビルド
		imageTag := fmt.Sprintf("%s:%s", conf.Docker.ImageName, timestamp)
		buildCmd := exec.Command("docker", "build", "-t", imageTag, conf.Docker.Context, "--no-cache")
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("ビルドに失敗: %w", err)
		}
	}

	fmt.Printf("Dockerイメージをビルドしました（タグ: %s）\n", timestamp)
	return nil
}

// SaveDockerImage は Docker イメージを tar.gz 形式で保存する関数
func SaveDockerImage(conf config.Config) error {
	fmt.Println("イメージを圧縮して保存中...")

	var imageTag string
	if conf.Docker.UseCompose {
		imageTag = fmt.Sprintf("%s_%s:%s", conf.Docker.ServiceName, conf.Docker.ServiceName, conf.Docker.Tag)
	} else {
		imageTag = fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag)
	}

	cmd := exec.Command("docker", "save", "-o", conf.Deploy.CompressedFile, imageTag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("圧縮に失敗: %w", err)
	}

	fmt.Printf("Dockerイメージ %s を %s に保存しました\n", imageTag, conf.Deploy.CompressedFile)
	os.Stdout.Sync()
	return nil
}

// TransferDockerImage は圧縮されたDockerイメージをリモートサーバーへ転送する関数
func TransferDockerImage(conf config.Config) error {
	remotePath := fmt.Sprintf("%s/%s", conf.Deploy.RemoteTempDir, conf.Deploy.CompressedFile)
	if conf.Docker.UseCompose {
		if err := TransferComposeFiles(conf); err != nil {
			return err
		}
	}
	return TransferFile(conf, conf.Deploy.CompressedFile, remotePath)
}

// RunRemoteContainer はリモートサーバーで古いコンテナを停止・削除し、新しいコンテナをデーモンモードで実行する関数
func RunRemoteContainer(conf config.Config) error {
	fmt.Println("\nリモートサーバーでコンテナを実行中...")

	// イメージのロード
	fmt.Println("1. Dockerイメージをロード中...")
	loadCmd := fmt.Sprintf("cd %s && docker load < %s", conf.Deploy.RemoteTempDir, conf.Deploy.CompressedFile)
	if err := ExecuteRemoteCommand(conf, loadCmd); err != nil {
		return fmt.Errorf("Dockerイメージのロードに失敗: %w", err)
	}

	if conf.Docker.UseCompose {
		// Docker Compose環境での実行
		stopCmd := fmt.Sprintf("cd %s && docker-compose -f %s down", conf.Deploy.RemoteTempDir, conf.Docker.ComposeFile)
		if err := ExecuteRemoteCommand(conf, stopCmd); err != nil {
			fmt.Printf("警告: 既存サービスの停止に失敗しました: %v\n", err)
		}

		// 新しいサービスの起動
		upCmd := fmt.Sprintf("cd %s && docker-compose -f %s up -d", conf.Deploy.RemoteTempDir, conf.Docker.ComposeFile)
		if err := ExecuteRemoteCommand(conf, upCmd); err != nil {
			return fmt.Errorf("Docker Composeサービスの起動に失敗: %w", err)
		}
	} else {
		// 従来の単一コンテナでの実行
		checkContainerCmd := fmt.Sprintf("docker ps -a --filter name=%s --format {{.Names}}", conf.Remote.ContainerName)
		containerOutput, err := executeRemoteCommandWithOutput(conf, checkContainerCmd)
		if err != nil {
			return fmt.Errorf("コンテナの確認に失敗: %w", err)
		}

		if containerOutput != "" {
			// 古いコンテナの停止と削除
			stopCmd := fmt.Sprintf("docker stop %s && docker rm %s", conf.Remote.ContainerName, conf.Remote.ContainerName)
			if err := ExecuteRemoteCommand(conf, stopCmd); err != nil {
				return fmt.Errorf("既存コンテナの停止・削除に失敗: %w", err)
			}
		}

		// 新しいコンテナの起動
		runCmd := fmt.Sprintf("docker run -d --name %s %s %s %s %s:%s",
			conf.Remote.ContainerName,
			formatPorts(conf.Remote.Ports),
			formatEnvs(conf.Remote.Environment),
			formatVolumes(conf.Remote.Volumes),
			conf.Docker.ImageName,
			conf.Docker.Tag,
		)
		if err := ExecuteRemoteCommand(conf, runCmd); err != nil {
			return fmt.Errorf("コンテナの起動に失敗: %w", err)
		}
	}

	fmt.Printf("新しいコンテナ/サービスの起動が完了しました\n")
	return nil
}

// RollbackToVersion は指定されたバージョンの Docker イメージでロールバックする関数
func RollbackToVersion(conf config.Config, version string) error {
	// デプロイ履歴から該当エントリを取得
	history, err := config.LoadHistory("config/history.toml")
	if err != nil {
		return err
	}
	entry, ok := history[version]
	if !ok {
		return fmt.Errorf("指定されたバージョン %s が見つかりません", version)
	}

	if entry.ComposeInfo.ServiceName != "" {
		// Docker Compose環境でのロールバック
		stopCmd := fmt.Sprintf("cd %s && docker-compose -f %s down", conf.Deploy.RemoteTempDir, conf.Docker.ComposeFile)
		if err := ExecuteRemoteCommand(conf, stopCmd); err != nil {
			fmt.Printf("警告: 既存サービスの停止に失敗しました: %v\n", err)
		}

		// docker-compose.yml内のイメージタグを更新（sed等を使用）
		updateCmd := fmt.Sprintf("cd %s && sed -i 's|image: .*|image: %s|' %s",
			conf.Deploy.RemoteTempDir,
			entry.Image,
			conf.Docker.ComposeFile,
		)
		if err := ExecuteRemoteCommand(conf, updateCmd); err != nil {
			return fmt.Errorf("compose設定の更新に失敗: %w", err)
		}

		// サービスの再起動
		upCmd := fmt.Sprintf("cd %s && docker-compose -f %s up -d", conf.Deploy.RemoteTempDir, conf.Docker.ComposeFile)
		return ExecuteRemoteCommand(conf, upCmd)
	} else {
		// 従来の単一コンテナでのロールバック
		stopCmd := fmt.Sprintf("docker stop %s && docker rm %s", conf.Remote.ContainerName, conf.Remote.ContainerName)
		runCmd := fmt.Sprintf("docker run -d --name %s %s %s %s %s",
			conf.Remote.ContainerName,
			formatPorts(conf.Remote.Ports),
			formatEnvs(conf.Remote.Environment),
			formatVolumes(conf.Remote.Volumes),
			entry.Image,
		)
		fullCmd := stopCmd + " && " + runCmd
		return ExecuteRemoteCommand(conf, fullCmd)
	}
}

// formatPorts はポート設定を "-p 80:80" のような形式に変換する
func formatPorts(ports []string) string {
	var result string
	for _, p := range ports {
		result += fmt.Sprintf(" -p %s", p)
	}
	return result
}

// formatEnvs は環境変数設定を "-e KEY=VALUE" のような形式に変換する
func formatEnvs(envs map[string]string) string {
	var result string
	for key, val := range envs {
		result += fmt.Sprintf(" -e %s=%s", key, val)
	}
	return result
}

// formatVolumes はボリューム設定を "-v host:container" の形式に変換する
func formatVolumes(volumes []string) string {
	var result string
	for _, v := range volumes {
		result += fmt.Sprintf(" -v %s", v)
	}
	return result
}

// executeRemoteCommandWithOutput は SSH を利用してリモートサーバー上でコマンドを実行し、その出力を返す関数
func executeRemoteCommandWithOutput(conf config.Config, command string) (string, error) {
	sshConfig, err := getSSHConfig(conf)
	if err != nil {
		return "", fmt.Errorf("SSH設定の取得に失敗: %w", err)
	}

	address := fmt.Sprintf("%s:%d", conf.SSH.Host, conf.SSH.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return "", fmt.Errorf("SSH接続に失敗: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("SSHセッションの作成に失敗: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("コマンドの実行に失敗: %w", err)
	}

	return string(output), nil
}
