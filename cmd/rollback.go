package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/linkalls/sailor/config"
	"github.com/linkalls/sailor/internal"

	"github.com/spf13/cobra"
)

// rollbackCmd は rollback コマンドの実装
var rollbackCmd = &cobra.Command{
	Use:   "rollback [version_identifier]",
	Short: "ロールバックを実行",
	Long:  "ロールバック可能なバージョンの一覧表示、または特定バージョンへのロールバックを実行します。",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 設定ファイル読み込み
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("カレントディレクトリの取得に失敗:", err)
			os.Exit(1)
		}
		configPath := filepath.Join(wd, "config/config.toml")
		conf, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Println("設定ファイルの読み込みに失敗:", err)
			os.Exit(1)
		}

// --list オプションで履歴一覧を表示
list, _ := cmd.Flags().GetBool("list")
if list {
if _, err := config.ShowDeployHistory(); err != nil {
fmt.Println("デプロイ履歴の表示に失敗:", err)
}
return
}

// バージョン識別子が未指定なら対話的に選択
if len(args) < 1 {
versions, err := config.ShowDeployHistory()
if err != nil {
fmt.Println("デプロイ履歴の表示に失敗:", err)
return
}

if len(versions) == 0 {
fmt.Println("ロールバック可能なバージョンがありません。")
return
}

// バージョン番号の入力受付
fmt.Print("\nロールバックするバージョン番号を入力してください: ")
var input string
fmt.Scanln(&input)

// 入力値の検証
validVersion := false
for _, v := range versions {
if v == input {
validVersion = true
break
}
}

if !validVersion {
fmt.Println("無効なバージョン番号です。")
return
}

args = []string{input}
}
		version := args[0]
		fmt.Printf("バージョン %s へのロールバックを実行中...\n", version)
		if err := internal.RollbackToVersion(conf, version); err != nil {
			fmt.Println("ロールバックに失敗:", err)
			return
		}
		fmt.Println("ロールバック完了！")
	},
}

func init() {
	rollbackCmd.Flags().BoolP("list", "l", false, "ロールバック可能なバージョンの一覧を表示")
}
