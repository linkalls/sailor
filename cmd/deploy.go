package cmd

import (
    "fmt"
    "os"
    "github.com/linkalls/sailor/config"
    "github.com/linkalls/sailor/internal"

    "github.com/spf13/cobra"
)

// deployCmd は deploy コマンドの実装
var deployCmd = &cobra.Command{
    Use:   "deploy",
    Short: "現在のブランチをデプロイ",
    Run: func(cmd *cobra.Command, args []string) {
        // Git状態のチェック
        if status, err := internal.CheckGitStatus(); !status {
            fmt.Println(err)
            return
        }

        // 設定ファイル読み込み
        conf, err := config.LoadConfig("config/config.toml")
        if err != nil {
            fmt.Println("設定ファイルの読み込みに失敗:", err)
            os.Exit(1)
        }

        // Git の現在のブランチが trigger_branch と一致しているか確認
        if !internal.CheckGitBranch(conf.Deploy.TriggerBranch) {
            fmt.Println("現在のブランチがトリガーブランチと一致しません。デプロイ中止。")
            return
        }

        // Dockerイメージのビルド
        fmt.Println("Dockerイメージをビルド中...")
        if err := internal.BuildDockerImage(conf); err != nil {
            fmt.Println("Dockerイメージのビルドに失敗:", err)
            return
        }

        // Dockerイメージの保存（圧縮）
        fmt.Println("Dockerイメージを圧縮中...")
        if err := internal.SaveDockerImage(conf); err != nil {
            fmt.Println("Dockerイメージの保存に失敗:", err)
            return
        }

        // ローカルの圧縮ファイルをリモートサーバーに転送
        fmt.Println("リモートサーバーへ転送中...")
        if err := internal.TransferFile(conf); err != nil {
            fmt.Println("ファイル転送に失敗:", err)
            return
        }

        // リモートサーバーでコンテナを実行（既存コンテナは停止・削除してから）
        fmt.Println("リモートサーバーでコンテナを実行中...")
        if err := internal.RunRemoteContainer(conf); err != nil {
            fmt.Println("コンテナの実行に失敗:", err)
            return
        }

        // デプロイ履歴を記録（TOML形式）
        if err := config.RecordDeployHistory(conf); err != nil {
            fmt.Println("デプロイ履歴の記録に失敗:", err)
        }

        fmt.Println("デプロイ完了！")
    },
}
