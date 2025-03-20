package main

import (
    "os"
    "github.com/linkalls/sailor/cmd"
)

func main() {
    // CLI のエントリーポイントを呼び出す
    if err := cmd.Execute(); err != nil {
        os.Exit(1)
    }
}
