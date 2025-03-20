package internal

import (
    "bytes"
    "fmt"
    "os/exec"
    "strings"
)

// CheckGitBranch は現在の Git ブランチが指定されたブランチか確認する関数
func CheckGitBranch(triggerBranch string) bool {
    // "git rev-parse --abbrev-ref HEAD" で現在のブランチ名を取得
    cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        return false
    }
    currentBranch := strings.TrimSpace(out.String())
    return currentBranch == triggerBranch
}

// CheckGitStatus はGitの状態をチェックする関数
func CheckGitStatus() (bool, error) {
    // 未コミットの変更があるかチェック
    cmd := exec.Command("git", "status", "--porcelain")
    output, err := cmd.Output()
    if err != nil {
        return false, fmt.Errorf("gitステータスの確認に失敗: %v", err)
    }
    
    // 出力が空でない場合は未コミットの変更がある
    if len(output) > 0 {
        return false, fmt.Errorf("未コミットの変更があります。先にコミットしてください")
    }
    
    return true, nil
}
