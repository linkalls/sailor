package cmd

import (
    "github.com/spf13/cobra"
)

// rootCmd は Sailor の基本コマンド
var rootCmd = &cobra.Command{
    Use:   "sailor",
    Short: "Sailor - シンプルなデプロイツール",
    Long:  "Sailorは、Dockerを用いた簡単なデプロイツールです。",
}

// Execute は CLI を実行する関数
func Execute() error {
    return rootCmd.Execute()
}

func init() {
    // 各サブコマンドを追加
    rootCmd.AddCommand(deployCmd)
    rootCmd.AddCommand(rollbackCmd)
    rootCmd.AddCommand(initCmd)
    rootCmd.AddCommand(configCmd)
    rootCmd.AddCommand(helpCmd)
}
