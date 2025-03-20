package internal

import (
    "fmt"
    "golang.org/x/crypto/ssh"
    "io/ioutil"
    "github.com/linkalls/sailor/config"
    "time"
)

// ExecuteRemoteCommand は SSH を利用してリモートサーバー上でコマンドを実行する関数
// 鍵認証またはパスワード認証のどちらにも対応
func ExecuteRemoteCommand(conf config.Config, command string) error {
    var authMethods []ssh.AuthMethod

    if conf.SSH.Password != "" {
        // パスワード認証を利用
        authMethods = append(authMethods, ssh.Password(conf.SSH.Password))
    } else if conf.SSH.PrivateKeyPath != "" {
        // 鍵認証を利用
        key, err := ioutil.ReadFile(conf.SSH.PrivateKeyPath)
        if err != nil {
            return err
        }
        signer, err := ssh.ParsePrivateKey(key)
        if err != nil {
            return err
        }
        authMethods = append(authMethods, ssh.PublicKeys(signer))
    } else {
        return fmt.Errorf("SSH認証情報が設定されていません")
    }

    sshConfig := &ssh.ClientConfig{
        User:            conf.SSH.User,
        Auth:            authMethods,
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
        Timeout:         5 * time.Second,
    }

    address := fmt.Sprintf("%s:%d", conf.SSH.Host, conf.SSH.Port)
    client, err := ssh.Dial("tcp", address, sshConfig)
    if err != nil {
        return err
    }
    defer client.Close()

    session, err := client.NewSession()
    if err != nil {
        return err
    }
    defer session.Close()

    return session.Run(command)
}
