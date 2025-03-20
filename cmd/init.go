package cmd

import (
    "fmt"
    "os"
    "github.com/linkalls/sailor/config"

    "github.com/spf13/cobra"
)

// initCmd は初期設定ファイル生成コマンド
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "デフォルトの設定ファイルを生成",
    Run: func(cmd *cobra.Command, args []string) {
        // 設定ファイル生成（既存ファイルがあれば上書きしないよう注意）
        if _, err := os.Stat("config/config.toml"); err == nil {
            fmt.Println("設定ファイルは既に存在します。")
            return
        }
        if err := config.GenerateDefaultConfig("config/config.toml"); err != nil {
            fmt.Println("設定ファイルの生成に失敗:", err)
            os.Exit(1)
        }
        fmt.Println("デフォルトの設定ファイルを生成しました: config/config.toml")
    },
}
