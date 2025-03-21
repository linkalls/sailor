package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/linkalls/sailor/config"
	"github.com/linkalls/sailor/internal"

	"github.com/spf13/cobra"
)

// deployCmd は deploy コマンドの実装
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "現在のブランチをデプロイ",
	Run: func(cmd *cobra.Command, args []string) {
		// カレントディレクトリの取得
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("カレントディレクトリの取得に失敗:", err)
			os.Exit(1)
		}

		// Git状態のチェック
		if status, err := internal.CheckGitStatus(); !status {
			fmt.Println(err)
			return
		}

		// 設定ファイル読み込み（カレントディレクトリからの相対パス）
		configPath := filepath.Join(wd, "config/config.toml")
		conf, err := config.LoadConfig(configPath)
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
		if err := internal.BuildDockerImage(&conf); err != nil {
			fmt.Printf("\nDockerイメージのビルドに失敗: %v\n", err)
			return
		}

		// Dockerイメージの保存（圧縮）
		if err := internal.SaveDockerImage(conf); err != nil {
			fmt.Printf("\nDockerイメージの保存に失敗: %v\n", err)
			return
		}

		fmt.Println("\nリモートサーバーへの転送を開始します...")

		// ローカルの圧縮ファイルをリモートサーバーに転送
		if err := internal.TransferDockerImage(conf); err != nil {
			fmt.Printf("\nファイル転送に失敗: %v\n", err)
			return
		}

		// リモートサーバーでコンテナを実行（既存コンテナは停止・削除してから）
		if err := internal.RunRemoteContainer(conf); err != nil {
			fmt.Printf("\nコンテナの実行に失敗: %v\n", err)
			return
		}

		// デプロイ履歴を記録（TOML形式）
		if err := config.RecordDeployHistory(conf); err != nil {
			fmt.Println("デプロイ履歴の記録に失敗:", err)
		}

		fmt.Println("デプロイ完了！")
	},
}
