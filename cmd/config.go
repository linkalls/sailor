package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/linkalls/sailor/config"

	"github.com/spf13/cobra"
)

// configCmd は設定内容表示コマンド
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "現在の設定ファイルの内容を表示",
	Run: func(cmd *cobra.Command, args []string) {
		// カレントディレクトリの取得
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("カレントディレクトリの取得に失敗:", err)
			os.Exit(1)
		}
		configPath := filepath.Join(wd, "config/config.toml")

		conf, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Println("設定ファイルの読み込みに失敗:", err)
			return
		}
		fmt.Printf("現在の設定:\n%+v\n", conf)
	},
}
