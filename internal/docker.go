package internal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/linkalls/sailor/config"
	"golang.org/x/crypto/ssh"
)

// BuildDockerImage は Docker イメージをビルドする関数
func BuildDockerImage(conf *config.Config) error {
	fmt.Println("Dockerイメージをビルド中...")

	// タイムスタンプベースのタグを生成
	timestamp := time.Now().Format("20060102150405")
	imageTag := fmt.Sprintf("%s:%s", conf.Docker.ImageName, timestamp)
	conf.Docker.Tag = timestamp

	// タイムスタンプタグでビルド
	buildCmd := exec.Command("docker", "build", "-t", imageTag, conf.Docker.Context, "--no-cache")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("ビルドに失敗: %w", err)
	}

	fmt.Printf("Dockerイメージをビルドしました（%s）\n", imageTag)

	// 設定にタイムスタンプタグを保存（後のロールバック用）
	conf.Docker.Tag = timestamp

	return nil
}

// SaveDockerImage は Docker イメージを tar.gz 形式で保存する関数
func SaveDockerImage(conf config.Config) error {
	fmt.Println("イメージを圧縮して保存中...")

	// タイムスタンプタグのイメージを保存（イメージ名:タグ の形式）
	imageTag := fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag)
	cmd := exec.Command("docker", "save", "-o", conf.Deploy.CompressedFile, imageTag)

	// 標準出力とエラー出力を直接表示
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// コマンドの実行と完了待機
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("圧縮に失敗: %w", err)
	}

	fmt.Printf("Dockerイメージ %s を %s に保存しました\n", imageTag, conf.Deploy.CompressedFile)

	// 標準出力をフラッシュ
	os.Stdout.Sync()
	return nil
}

// min は2つの整数の小さい方を返す
// func min(a, b int) int {
//     if a < b {
//         return a
//     }
//     return b
// }

// TransferDockerImage は圧縮されたDockerイメージをリモートサーバーへ転送する関数
func TransferDockerImage(conf config.Config) error {
	remotePath := fmt.Sprintf("%s/%s", conf.Deploy.RemoteTempDir, conf.Deploy.CompressedFile)
	// fmt.Printf("%s/%s", conf.Deploy.RemoteTempDir, conf.Deploy.CompressedFile)
	return TransferFile(conf, conf.Deploy.CompressedFile, remotePath)
}

// RunRemoteContainer はリモートサーバーで古いコンテナを停止・削除し、新しいコンテナをデーモンモードで実行する関数
func RunRemoteContainer(conf config.Config) error {
	fmt.Println("\nリモートサーバーでコンテナを実行中...")

	// イメージのロードとタグ付け
	fmt.Println("1. Dockerイメージをロード中...")
	loadCmd := fmt.Sprintf("cd %s && docker load < %s", conf.Deploy.RemoteTempDir, conf.Deploy.CompressedFile)
	fmt.Println(loadCmd)
	if err := ExecuteRemoteCommand(conf, loadCmd); err != nil {
		return fmt.Errorf("Dockerイメージのロードに失敗: %w", err)
	}

	// タイムスタンプタグでイメージが正しくロードされたか確認
	checkTagCmd := fmt.Sprintf("docker images %s:%s --format '{{.Repository}}:{{.Tag}}'", conf.Docker.ImageName, conf.Docker.Tag)
	tagOutput, err := executeRemoteCommandWithOutput(conf, checkTagCmd)
	if err != nil || strings.TrimSpace(tagOutput) == "" {
		return fmt.Errorf("イメージのロード確認に失敗: %w", err)
	}
	fmt.Printf("イメージ %s のロードが完了しました\n", strings.TrimSpace(tagOutput))

	// 既存コンテナの確認
	fmt.Println("2. 既存コンテナの確認中...")
	checkContainerCmd := fmt.Sprintf("docker ps -a --filter name=%s --format {{.Names}}", conf.Remote.ContainerName)
	containerOutput, err := executeRemoteCommandWithOutput(conf, checkContainerCmd)
	if err != nil {
		return fmt.Errorf("コンテナの確認に失敗: %w", err)
	}

	// 既存コンテナが存在する場合のみ停止・削除を実行
	if containerOutput != "" {
		// 古いコンテナの停止
		fmt.Println("3. 既存コンテナを停止中...")
		stopCmd := fmt.Sprintf("docker stop %s", conf.Remote.ContainerName)
		if err := ExecuteRemoteCommand(conf, stopCmd); err != nil {
			return fmt.Errorf("コンテナの停止に失敗: %w", err)
		}

		// 古いコンテナの削除
		fmt.Println("4. 既存コンテナを削除中...")
		rmCmd := fmt.Sprintf("docker rm %s", conf.Remote.ContainerName)
		if err := ExecuteRemoteCommand(conf, rmCmd); err != nil {
			return fmt.Errorf("コンテナの削除に失敗: %w", err)
		}
	} else {
		fmt.Printf("既存の %s コンテナは存在しないためスキップします\n", conf.Remote.ContainerName)
	}

	// 新しいコンテナの起動（-d でデーモンモード）
	fmt.Println("6. 新しいコンテナを起動中...")
	runCmd := fmt.Sprintf("docker run -d --name %s %s %s %s %s:%s",
		conf.Remote.ContainerName,
		formatPorts(conf.Remote.Ports),
		formatEnvs(conf.Remote.Environment),
		formatVolumes(conf.Remote.Volumes),
		conf.Docker.ImageName,
		conf.Docker.Tag, // タイムスタンプタグを直接使用
	)
	if err := ExecuteRemoteCommand(conf, runCmd); err != nil {
		return fmt.Errorf("コンテナの起動に失敗: %w", err)
	}
	fmt.Printf("新しいコンテナ（イメージ: %s:%s）の起動が完了しました\n", conf.Docker.ImageName, conf.Docker.Tag)

	return nil
}

// executeRemoteCommandWithOutput は SSH を利用してリモートサーバー上でコマンドを実行し、その出力を返す関数
func executeRemoteCommandWithOutput(conf config.Config, command string) (string, error) {
	// SSH設定の取得
	sshConfig, err := getSSHConfig(conf)
	if err != nil {
		return "", fmt.Errorf("SSH設定の取得に失敗: %w", err)
	}

	// SSHクライアントの作成
	address := fmt.Sprintf("%s:%d", conf.SSH.Host, conf.SSH.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return "", fmt.Errorf("SSH接続に失敗: %w", err)
	}
	defer client.Close()

	// セッションの作成
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("SSHセッションの作成に失敗: %w", err)
	}
	defer session.Close()

	// コマンドの実行と出力の取得
	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("コマンドの実行に失敗: %w", err)
	}

	return string(output), nil
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
	// 古いコンテナの停止・削除と対象イメージでコンテナ起動
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
