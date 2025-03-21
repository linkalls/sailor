package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/linkalls/sailor/config"

	"github.com/spf13/cobra"
)

// initCmd は初期設定ファイル生成コマンド
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "デフォルトの設定ファイルを生成",
	Run: func(cmd *cobra.Command, args []string) {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("カレントディレクトリの取得に失敗:", err)
			os.Exit(1)
		}
		// 設定ファイル生成（既存ファイルがあれば上書きしないよう注意）
		configPath := filepath.Join(wd, "config/config.toml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Println("設定ファイルは既に存在します。")
			return
		}
		if err := config.GenerateDefaultConfig("config/config.toml"); err != nil {
			fmt.Println("設定ファイルの生成に失敗:", err)
			os.Exit(1)
		}
		fmt.Println("デフォルトの設定ファイルを生成しました: config/config.toml")

		// Dockerfileの生成（存在しない場合のみ）
		if _, err := os.Stat("Dockerfile"); os.IsNotExist(err) {
			if err := config.GenerateDefaultDockerfile("Dockerfile"); err != nil {
				fmt.Println("Dockerfileの生成に失敗:", err)
				os.Exit(1)
			}
			fmt.Println("デフォルトのDockerfileを生成しました: Dockerfile")
		}
	},
}
