package internal

import (
    "fmt"
    "os/exec"
    "github.com/linkalls/sailor/config"
)

// BuildDockerImage は Docker イメージをビルドする関数
func BuildDockerImage(conf config.Config) error {
    // docker build -t image_name:tag context
   	cmd := exec.Command("docker", "build", "-t", fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag), conf.Docker.Context)
    // コマンドの出力はここでは省略
    return cmd.Run()
}

// SaveDockerImage は Docker イメージを tar.gz 形式で保存する関数
func SaveDockerImage(conf config.Config) error {
    // docker save -o compressed_file image_name:tag
    cmd := exec.Command("docker", "save", "-o", conf.Deploy.CompressedFile, fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag))
    return cmd.Run()
}

// TransferFile はローカルの圧縮ファイルをリモートサーバーへ転送する関数
// パスワード認証の場合は sshpass を利用（sshpass が必要）
// 鍵認証の場合は -i オプションを利用
func TransferFile(conf config.Config) error {
    remotePath := fmt.Sprintf("%s/%s", conf.Deploy.RemoteTempDir, conf.Deploy.CompressedFile)
		fmt.Println(remotePath)
    if conf.SSH.Password != "" {
        // パスワード認証（sshpass 使用）
        cmd := exec.Command("sshpass", "-p", conf.SSH.Password, "scp", "-P", fmt.Sprintf("%d", conf.SSH.Port), conf.Deploy.CompressedFile, fmt.Sprintf("%s@%s:%s", conf.SSH.User, conf.SSH.Host, conf.Deploy.RemoteTempDir))
        return cmd.Run()
    } else {
        // 鍵認証の場合
        cmd := exec.Command("scp", "-P", fmt.Sprintf("%d", conf.SSH.Port), "-i", conf.SSH.PrivateKeyPath, conf.Deploy.CompressedFile, fmt.Sprintf("%s@%s:%s", conf.SSH.User, conf.SSH.Host, conf.Deploy.RemoteTempDir))
        return cmd.Run()
    }
}

// RunRemoteContainer はリモートサーバーで古いコンテナを停止・削除し、新しいコンテナをデーモンモードで実行する関数
func RunRemoteContainer(conf config.Config) error {
    // 古いコンテナの停止と削除
    stopCmd := fmt.Sprintf("docker stop %s && docker rm %s", conf.Remote.ContainerName, conf.Remote.ContainerName)
    // 新しいコンテナの起動（-d でデーモンモード）
    runCmd := fmt.Sprintf("docker run -d --name %s %s %s %s %s",
        conf.Remote.ContainerName,
        formatPorts(conf.Remote.Ports),
        formatEnvs(conf.Remote.Environment),
        formatVolumes(conf.Remote.Volumes),
        fmt.Sprintf("%s:%s", conf.Docker.ImageName, conf.Docker.Tag),
    )
    fullCmd := stopCmd + " && " + runCmd
    return ExecuteRemoteCommand(conf, fullCmd)
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
