package internal

import (
"bufio"
"bytes"
"fmt"
"io"
"os"
"path/filepath"
"time"

	"github.com/linkalls/sailor/config"
	"golang.org/x/crypto/ssh"
)

// getSSHConfig はSSH接続の設定を生成する関数
func getSSHConfig(conf config.Config) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	if conf.SSH.Password != "" {
		// パスワード認証を利用
		authMethods = append(authMethods, ssh.Password(conf.SSH.Password))
	} else if conf.SSH.PrivateKeyPath != "" {
		// 鍵認証を利用
		key, err := os.ReadFile(conf.SSH.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("SSHキーの読み込みに失敗: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("SSHキーの解析に失敗: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		return nil, fmt.Errorf("SSH認証情報が設定されていません")
	}

	// SSH設定を生成
	config := &ssh.ClientConfig{
		User:            conf.SSH.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.FixedHostKey(nil), // デフォルトではnil、後で上書き
		Timeout:         10 * time.Minute,      // タイムアウトを10分に延長
	}

	// パスワード認証の場合は、既知のホストキーを使用
	if conf.SSH.Password != "" {
		config.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	return config, nil
}

// TransferFile は指定されたファイルをSSH経由で転送する関数
func TransferFile(conf config.Config, localPath string, remotePath string) error {
    // SSH設定の取得
    sshConfig, err := getSSHConfig(conf)
    if err != nil {
        return fmt.Errorf("SSH設定の取得に失敗: %w", err)
    }

    // SSHクライアントの作成
    address := fmt.Sprintf("%s:%d", conf.SSH.Host, conf.SSH.Port)
    client, err := ssh.Dial("tcp", address, sshConfig)
    if err != nil {
        return fmt.Errorf("SSH接続に失敗: %w", err)
    }
    defer client.Close()

    // セッションの作成
    session, err := client.NewSession()
    if err != nil {
        return fmt.Errorf("SSHセッションの作成に失敗: %w", err)
    }
    defer session.Close()

    // ローカルファイルを開く
    localFile, err := os.Open(localPath)
    if err != nil {
        return fmt.Errorf("ローカルファイルのオープンに失敗: %w", err)
    }
    defer localFile.Close()

    // リモートファイルのパスを作成
    remoteDir := filepath.Dir(remotePath)
    // リモートディレクトリの作成とSCPコマンドの存在確認
    checkCmd := fmt.Sprintf("mkdir -p %s && which scp", remoteDir)
    err = executeCommand(client, checkCmd)
    if err != nil {
        return fmt.Errorf("リモートディレクトリの作成またはSCPコマンドの確認に失敗: %w", err)
    }

    // ファイルの情報を取得
    fileInfo, err := localFile.Stat()
    if err != nil {
        return fmt.Errorf("ファイル情報の取得に失敗: %w", err)
    }

    // セッションの標準入力と出力を設定（バッファ付き）
    stdin, err := session.StdinPipe()
    if err != nil {
        return fmt.Errorf("入力パイプの作成に失敗: %w", err)
    }
    w := bufio.NewWriter(stdin)

    stdout, err := session.StdoutPipe()
    if err != nil {
        return fmt.Errorf("出力パイプの作成に失敗: %w", err)
    }
    r := bufio.NewReader(stdout)

    // エラー出力の取得用に設定
    var stderrBuf bytes.Buffer
    session.Stderr = &stderrBuf

    // SCPコマンドの実行（絶対パスを使用）
    remoteCmd := fmt.Sprintf("/usr/bin/scp -t %s", remotePath)
    if err := session.Start(remoteCmd); err != nil {
        return fmt.Errorf("SCPコマンドの実行に失敗: %w\nStderr: %s", err, stderrBuf.String())
    }

    fmt.Printf("%s を %s に転送中...\n", localPath, remotePath)

    // 初期ACKの確認（リトライ付き）
    if err := retryAck(r, "初期確認"); err != nil {
        return fmt.Errorf("%w\nStderr: %s", err, stderrBuf.String())
    }

    // ファイル情報を送信
    command := fmt.Sprintf("C%04o %d %s\n", fileInfo.Mode()&0777, fileInfo.Size(), filepath.Base(remotePath))
    _, err = w.Write([]byte(command))
    if err != nil {
        return fmt.Errorf("ファイル情報の送信に失敗: %w", err)
    }
    if err := w.Flush(); err != nil {  // コマンド送信後にフラッシュ
        return fmt.Errorf("コマンドバッファのフラッシュに失敗: %w", err)
    }

    // ACKの確認（リトライ付き）
    if err := retryAck(r, "ファイル情報の確認"); err != nil {
        return fmt.Errorf("%w\nStderr: %s", err, stderrBuf.String())
    }

    // バッファ付きの転送と進捗表示
    buffer := make([]byte, 1024*1024) // 1MB バッファ
    totalSize := fileInfo.Size()
    transferred := int64(0)
    lastUpdateTime := time.Now()
    updateInterval := 100 * time.Millisecond

    description := fmt.Sprintf("%s -> %s", filepath.Base(localPath), remotePath)
    fmt.Printf("\n%s の転送を開始します\n", description)

    for {
        n, err := localFile.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("ファイル読み込みに失敗: %w", err)
        }

        _, err = w.Write(buffer[:n])
        if err != nil {
            return fmt.Errorf("ファイル書き込みに失敗: %w", err)
        }

        // 定期的なフラッシュを追加
        if err := w.Flush(); err != nil {
            return fmt.Errorf("転送中のバッファフラッシュに失敗: %w", err)
        }

        transferred += int64(n)

        // 100ミリ秒ごとに表示を更新
        if time.Since(lastUpdateTime) >= updateInterval {
            percentage := float64(transferred) / float64(totalSize) * 100
            mbTransferred := float64(transferred) / 1024 / 1024
            mbTotal := float64(totalSize) / 1024 / 1024
            fmt.Printf("\r転送中: %.1f%% 完了 (%.1f/%.1f MB)", percentage, mbTransferred, mbTotal)
            lastUpdateTime = time.Now()
        }
    }

    // 最終的な進捗を表示
    percentage := float64(transferred) / float64(totalSize) * 100
    mbTransferred := float64(transferred) / 1024 / 1024
    mbTotal := float64(totalSize) / 1024 / 1024
    fmt.Printf("\r転送中: %.1f%% 完了 (%.1f/%.1f MB)\n", percentage, mbTransferred, mbTotal)
    fmt.Printf("%s の転送が完了しました\n", description)

    // バッファをフラッシュ
    if err := w.Flush(); err != nil {
        return fmt.Errorf("最終バッファのフラッシュに失敗: %w", err)
    }

    // 終了シグナルを送信
    if _, err := w.Write([]byte{0}); err != nil {
        return fmt.Errorf("終了シグナルの送信に失敗: %w", err)
    }

    // 最後のフラッシュ
    if err := w.Flush(); err != nil {
        return fmt.Errorf("最終バッファのフラッシュに失敗: %w", err)
    }

    // 入力パイプを閉じる
    if err := stdin.Close(); err != nil {
        return fmt.Errorf("入力パイプの終了に失敗: %w", err)
    }

    // 最終ACKの確認（リトライ付き）
    if err := retryAck(r, "ファイル転送の確認"); err != nil {
        return fmt.Errorf("%w\nStderr: %s", err, stderrBuf.String())
    }

    // セッションの完了を待機
    if err := session.Wait(); err != nil {
        return fmt.Errorf("ファイル転送の完了待機中にエラー: %w\nStderr: %s", err, stderrBuf.String())
    }

    return nil
}

// ExecuteRemoteCommand は SSH を利用してリモートサーバー上でコマンドを実行する関数
func ExecuteRemoteCommand(conf config.Config, command string) error {
	// SSH設定の取得
	sshConfig, err := getSSHConfig(conf)
	if err != nil {
		return fmt.Errorf("SSH設定の取得に失敗: %w", err)
	}

	// SSHクライアントの作成
	address := fmt.Sprintf("%s:%d", conf.SSH.Host, conf.SSH.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH接続に失敗: %w", err)
	}
	defer client.Close()

	// セッションの作成と実行
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("SSHセッションの作成に失敗: %w", err)
	}
	defer session.Close()

	// 標準出力とエラー出力を設定
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	return session.Run(command)
}

// executeCommand は SSH クライアントを使用してコマンドを実行するヘルパー関数
func executeCommand(client *ssh.Client, command string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 標準出力とエラー出力を設定
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	return session.Run(command)
}

// retryAck はACK確認を複数回試行する関数
func retryAck(r *bufio.Reader, phase string) error {
	var lastErr error
	for retry := 0; retry < 3; retry++ {
		if err := checkAck(r); err == nil {
			return nil
		} else {
			lastErr = err
			time.Sleep(time.Second * time.Duration(retry+1))
		}
	}
	return fmt.Errorf("%sに失敗: %w", phase, lastErr)
}

// checkAck はSCPプロトコルの応答を確認する関数
func checkAck(r io.Reader) error {
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("予期せぬEOF: ACK応答が受信できません")
			}
			return fmt.Errorf("ACK読み取りエラー: %w", err)
		}
		if n < 1 {
			continue // 読み取りを再試行
		}

		switch buf[0] {
		case 0:
			return nil
		case 1, 2:
			var errorStr string
			scanner := bufio.NewScanner(r)
			if scanner.Scan() {
				errorStr = scanner.Text()
			}
			return fmt.Errorf("サーバーエラー応答 (コード %d): %s", buf[0], errorStr)
		default:
			return fmt.Errorf("不正な応答コード: %d", buf[0])
		}
	}
}
