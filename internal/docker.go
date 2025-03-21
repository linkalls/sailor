package internal

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/linkalls/sailor/config"
)

// BuildDockerImage は Docker イメージをビルドする関数
func BuildDockerImage(conf config.Config) error {
	fmt.Println("Dockerイメージをビルド中...")

	// docker build -t image_name:tag context
	cmd := exec.Command("docker", "build", "-t", fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag), "--no-cache", conf.Docker.Context)

	// 標準出力とエラー出力を直接表示
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// コマンドの実行と完了待機
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ビルドに失敗: %w", err)
	}

	fmt.Println("Dockerイメージのビルドが完了しました")
	return nil
}

// SaveDockerImage は Docker イメージを tar.gz 形式で保存する関数
func SaveDockerImage(conf config.Config) error {
	fmt.Println("イメージを圧縮して保存中...")

	// docker save -o compressed_file image_name:tag
	cmd := exec.Command("docker", "save", "-o", conf.Deploy.CompressedFile, fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag))

	// 標準出力とエラー出力を直接表示
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// コマンドの実行と完了待機
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("圧縮に失敗: %w", err)
	}

	fmt.Printf("Dockerイメージを %s に保存しました\n", conf.Deploy.CompressedFile)

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

	// イメージの解凍とロード
	fmt.Println("1. Dockerイメージをロード中...")
	loadCmd := fmt.Sprintf("cd %s && docker load < %s", conf.Deploy.RemoteTempDir, conf.Deploy.CompressedFile)
	if err := ExecuteRemoteCommand(conf, loadCmd); err != nil {
		return fmt.Errorf("Dockerイメージのロードに失敗: %w", err)
	}
	fmt.Println("イメージのロードが完了しました")

	// 古いコンテナの停止
	fmt.Println("2. 既存コンテナを停止中...")
	stopCmd := fmt.Sprintf("docker stop %s", conf.Remote.ContainerName)
	if err := ExecuteRemoteCommand(conf, stopCmd); err != nil {
		fmt.Printf("注意: 古いコンテナの停止中にエラー: %v\n", err)
	}

	// 古いコンテナの削除
	fmt.Println("3. 既存コンテナを削除中...")
	rmCmd := fmt.Sprintf("docker rm %s || true", conf.Remote.ContainerName)
	if err := ExecuteRemoteCommand(conf, rmCmd); err != nil {
		fmt.Printf("注意: 古いコンテナの削除中にエラー: %v\n", err)
	}

	// 新しいコンテナの起動（-d でデーモンモード）
	fmt.Println("4. 新しいコンテナを起動中...")
	runCmd := fmt.Sprintf("docker run -d --name %s %s %s %s %s",
		conf.Remote.ContainerName,
		formatPorts(conf.Remote.Ports),
		formatEnvs(conf.Remote.Environment),
		formatVolumes(conf.Remote.Volumes),
		fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag),
	)
	if err := ExecuteRemoteCommand(conf, runCmd); err != nil {
		return fmt.Errorf("コンテナの起動に失敗: %w", err)
	}
	fmt.Println("新しいコンテナの起動が完了しました")

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
