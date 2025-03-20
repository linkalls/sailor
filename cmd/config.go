package cmd

import (
    "fmt"
    "github.com/linkalls/sailor/config"

    "github.com/spf13/cobra"
)

// configCmd は設定内容表示コマンド
var configCmd = &cobra.Command{
    Use:   "config",
    Short: "現在の設定ファイルの内容を表示",
    Run: func(cmd *cobra.Command, args []string) {
        conf, err := config.LoadConfig("config/config.toml")
        if err != nil {
            fmt.Println("設定ファイルの読み込みに失敗:", err)
            return
        }
        fmt.Printf("現在の設定:\n%+v\n", conf)
    },
}
