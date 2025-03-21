package internal

import (
    "bytes"
    "fmt"
    "os/exec"
    "strings"
    "time"
)

// ProgressBar は進捗状態を表示するための構造体
type ProgressBar struct {
    description string
    width       int
    current     int
    total       int
    lastUpdate  time.Time
}

// NewProgressBar は新しいProgressBarを作成
func NewProgressBar(description string, total int) *ProgressBar {
    p := &ProgressBar{
        description: description,
        width:      20,
        total:      total,
        lastUpdate: time.Now(),
    }
    // 初期状態を表示
    p.Update(0)
    return p
}

// Update は進捗を更新
func (p *ProgressBar) Update(current int) {
    p.current = current
    
    // プログレスバーの作成
    progress := float64(current) / float64(p.total)
    completed := int(progress * float64(p.width))
    
    // バーの部分を構築
    bar := make([]byte, p.width)
    for i := 0; i < p.width; i++ {
        if i < completed {
            bar[i] = '='
        } else {
            bar[i] = ' '
        }
    }
    
    // 進捗状況を表示（改行なし、バイト数含む）
    fmt.Printf("\r[%-20s] %s (%d/%d バイト)", 
        string(bar), 
        p.description, 
        current,
        p.total)
}

// Complete は進捗を完了状態に
func (p *ProgressBar) Complete() {
    bar := strings.Repeat("=", p.width)
    // 完了メッセージを表示
    fmt.Printf("\r[%s] %s 完了\n", bar, p.description)
}

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
