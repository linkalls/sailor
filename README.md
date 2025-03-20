# Sailor

Dockerコンテナを使用したシンプルなデプロイツール

## インストール

```bash
go install github.com/linkalls/sailor@latest
```

## 使用方法

### 1. 初期設定

プロジェクトディレクトリで以下のコマンドを実行して設定ファイルを生成します：

```bash
sailor init
```

これにより、`config/config.toml` が作成されます。

### 2. 設定ファイルの編集

`config/config.toml` を編集して、以下の項目を設定します：

```toml
[deploy]
trigger_branch = "main"  # デプロイを実行するブランチ
image_name = "app"       # Dockerイメージ名
container_name = "app"   # コンテナ名

[remote]
host = "example.com"     # デプロイ先サーバーのホスト名
user = "username"        # SSHユーザー名
port = "22"             # SSHポート
```

### 3. デプロイの実行

デプロイを実行する前に、必ず変更をコミットしてください：

```bash
# 変更をステージングに追加
git add .

# 変更をコミット
git commit -m "デプロイする変更の説明"
```

その後、以下のコマンドでデプロイを実行します：

```bash
sailor deploy
```

注意事項：
- 未コミットの変更がある場合、デプロイは実行されません
- トリガーブランチ（デフォルトではmain）以外のブランチからはデプロイできません

### 4. ロールバック

問題が発生した場合は、以下のコマンドで前回のバージョンに戻すことができます：

```bash
sailor rollback
```

## エラーメッセージについて

### "未コミットの変更があります。先にコミットしてください"
- 原因：ローカルに未コミットの変更が存在する
- 対処：`git add .` と `git commit` を実行して変更をコミットしてください

### "現在のブランチがトリガーブランチと一致しません"
- 原因：config.tomlで指定したトリガーブランチ以外からデプロイしようとした
- 対処：トリガーブランチに切り替えるか、config.tomlの設定を変更してください

## ライセンス

MIT License
