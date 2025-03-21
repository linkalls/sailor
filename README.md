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

#### Docker Compose を使用する場合

```toml
[docker]
use_compose = true                  # Docker Compose使用
compose_file = "docker-compose.yml" # compose設定ファイルのパス
service_name = "app"               # 対象のサービス名
compose_env_file = ".env"          # 環境変数ファイル

[compose]
env_files = [".env", ".env.prod"]  # 環境変数ファイル群
extra_files = [                    # 追加で転送が必要なファイル
    "nginx.conf",
    "mysql/init.sql"
]
target_env = "production"          # ビルド/デプロイ時の環境指定

[deploy]
trigger_branch = "main"            # デプロイを実行するブランチ
compressed_file = "deploy.tar.gz"  # 圧縮ファイル名
remote_temp_dir = "~/tmp"          # リモートの一時ディレクトリ

[ssh]
host = "example.com"               # デプロイ先サーバーのホスト名
user = "username"                  # SSHユーザー名
port = 22                         # SSHポート
```

#### 単一のDockerfileを使用する場合

```toml
[docker]
use_compose = false               # Docker Compose不使用
dockerfile = "Dockerfile"         # Dockerfileのパス
context = "./"                   # ビルドコンテキストのパス
image_name = "app"               # イメージ名
tag = "latest"                   # タグ名

[remote]
container_name = "app_container" # コンテナ名
ports = ["80:80"]               # ポートマッピング
environment = {                 # 環境変数
    "DATABASE_URL": "postgres://localhost:5432/db",
    "API_KEY": "your-api-key"
}
volumes = ["/data:/app/data"]   # ボリュームマウント

[deploy]
trigger_branch = "main"
compressed_file = "deploy.tar.gz"
remote_temp_dir = "~/tmp"

[ssh]
host = "example.com"
user = "username"
port = 22
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

デプロイ時の動作：
- Docker Compose使用時:
  1. docker-compose buildでイメージをビルド
  2. compose.yml、環境変数ファイル、追加ファイルを転送
  3. リモートサーバーでサービスを再起動

- Dockerfile使用時:
  1. docker buildでイメージをビルド
  2. イメージを転送
  3. リモートサーバーでコンテナを再起動

注意事項：
- 未コミットの変更がある場合、デプロイは実行されません
- トリガーブランチ（デフォルトではmain）以外のブランチからはデプロイできません

### 4. ロールバック

デプロイ履歴を確認するには以下のコマンドを使用します：

```bash
sailor rollback --list
```

これにより以下の情報が表示されます：
- バージョン番号
- コミットハッシュとメッセージ
- デプロイされたDockerイメージ
- デプロイ日時
- Docker Compose情報（使用時）
  - サービス名
  - 環境変数ファイル
  - 対象環境

特定のバージョンにロールバックするには：

```bash
# バージョン番号を指定してロールバック
sailor rollback <version>

# または、対話的に選択
sailor rollback
```

## エラーメッセージについて

### "未コミットの変更があります。先にコミットしてください"
- 原因：ローカルに未コミットの変更が存在する
- 対処：`git add .` と `git commit` を実行して変更をコミットしてください

### "現在のブランチがトリガーブランチと一致しません"
- 原因：config.tomlで指定したトリガーブランチ以外からデプロイしようとした
- 対処：トリガーブランチに切り替えるか、config.tomlの設定を変更してください

### "Docker Composeのビルドに失敗"
- 原因：docker-compose.ymlの設定エラーまたは依存関係の問題
- 対処：エラーメッセージを確認し、compose設定を修正してください

## ライセンス

MIT License
