package cmd

import (
    "fmt"

    "github.com/spf13/cobra"
)

// helpCmd は help コマンドの実装
var helpCmd = &cobra.Command{
    Use:   "help",
    Short: "コマンドの一覧や使い方を表示",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Sailor コマンド一覧:")
        fmt.Println("  init         - 設定ファイルの雛形を生成")
        fmt.Println("  deploy       - デプロイ処理を実行")
        fmt.Println("  rollback     - ロールバック処理を実行 (rollback --list で一覧表示)")
        fmt.Println("  config       - 現在の設定ファイルの内容を表示")
        fmt.Println("  help         - コマンドの使い方を表示")
    },
}
