# デプロイ・環境構成

## リポジトリ構成

モノレポ。`backend`(Go)と `frontend`(Next.js)を本リポジトリで一括管理する。

## 環境

| 環境 | ドメイン | 用途 | デプロイ契機 |
| --- | --- | --- | --- |
| 開発 (dev) | `dev.mc-werewolf.com` | 開発者のみが確認できる検証用環境 | `main` ブランチへの変更(push) |
| 本番 (prod) | `mc-werewolf.com` | 一般公開環境 | `main` 上でのtag作成 |

サブドメインは `dev.mc-werewolf.com` の1つのみを採用する。

## サーバー

- dev/prod は同一の物理サーバー上で稼働するが、環境同士は完全に分離する
  - DB(PostgreSQL)、backend、frontend いずれも dev/prod で別々のコンテナ・別々のデータとして稼働させ、共有しない
  - 具体的には docker compose の projectを dev/prod で分ける(例: `docker compose -p ww-dev ...` / `docker compose -p ww-prod ...`)ことで、コンテナ名・ネットワーク・named volume(PostgreSQLのデータ含む)を自動的に分離する
  - dev/prodそれぞれ専用の環境変数ファイル(DB名・ユーザー・パスワード・各ポート等)を用いる。値そのものを同じにしない
- ホスト名/IP・SSHユーザー等の接続情報は本リポジトリにコミットせず、GitHub Actions の Secrets / Variables で管理する
- SSH接続は主に初期セットアップ(Docker/Docker Composeのインストールなど、GitHub Actionsが動く前提を整えるための一度きりの作業)を想定したものであり、それ以降の継続的な運用は下記「運用方針」の通り GitHub Actions 経由で行う

## 運用方針

- サーバーへのログイン・手動での設定変更は行わない
- サーバー上の状態はすべて本リポジトリのdocker compose定義から生成し、変更は GitHub Actions 経由でのみ行う(Infrastructure as Code的な運用)
- Caddyも他サービス(backend/frontend/postgres)と同様に docker compose で管理するコンテナとしてサーバー上に配置する(ホストへの直接インストールはしない)

## デプロイフロー

1. GitHub Actions 上でCI(ビルド等)を実行する
2. Docker Hub にイメージをpushする(backend / frontend それぞれ)
3. サーバー側は基本的にDocker Hubからイメージをpull & 起動するのみで完結させる(ソースのやり取りはしない)
4. `main` へのpush → devへ自動デプロイ
5. `main` 上でのtag作成 → prodへ自動デプロイ

## GitHub Actions (CI/CD)

ワークフロー: [`.github/workflows/deploy.yml`](../.github/workflows/deploy.yml)

1. **build-and-push ジョブ**
   - backend / frontend それぞれの Docker イメージをビルドし、Docker Hub の `<DOCKERHUB_USERNAME>/mc-werewolf-backend` / `<DOCKERHUB_USERNAME>/mc-werewolf-frontend` にpush
   - イメージタグ: `main` へのpush時は `dev`、`v*.*.*` タグ作成時はそのタグ名(例 `v1.0.0`)をそのままタグとして使用
2. **deploy ジョブ**(`needs: build-and-push`)
   - デプロイ専用の [`deploy/docker-compose.yml`](../deploy/docker-compose.yml)(`build:` ではなく `image:` 参照。ローカル開発用のルート `docker-compose.yml` とは別ファイル)を SCP でサーバー上の `/opt/werewolf/<env>/` に配置
   - SSH接続し、環境ごとの `.env`(ポート・DB認証情報・イメージタグ等)をSecretsから生成した上で、`docker compose -p ww-<env> --env-file .env pull && up -d --wait --wait-timeout 90` を実行
   - `<env>` は `dev` / `prod`。docker composeの **project名を分ける**ことで、コンテナ・ネットワーク・named volume(DBデータ含む)を完全に分離する
   - `--wait` は各サービスの healthcheck が通るまでジョブを待たせ、タイムアウト・失敗時は非ゼロ終了でジョブごと失敗させる(後述「堅牢性」参照)
3. **deploy-caddy ジョブ**(`needs: deploy`)
   - [`deploy/caddy/docker-compose.yml`](../deploy/caddy/docker-compose.yml) / [`Caddyfile`](../deploy/caddy/Caddyfile) を SCP でサーバー上の `/opt/werewolf/caddy/` に配置
   - SSH接続し、Basic認証用の `.env` をSecretsから生成した上で、`docker compose -p ww-caddy --env-file .env pull && up -d --wait --wait-timeout 60` を実行
   - dev/prodどちらのpushでも毎回実行される(Caddyは環境共有のため)。前述の通り、初回はdev/prod両方のデプロイが完了済みである必要がある

### 必要なGitHub Secrets

| Secret | 用途 |
| --- | --- |
| `DOCKERHUB_USERNAME` | Docker Hubログイン、イメージの名前空間 |
| `DOCKERHUB_TOKEN` | Docker Hubログイン用アクセストークン |
| `DEPLOY_HOST` | デプロイ先サーバーのホスト名/IP |
| `DEPLOY_USER` | デプロイ先サーバーのSSHユーザー |
| `DEPLOY_SSH_KEY` | デプロイ先サーバーへのSSH秘密鍵 |
| `DEV_POSTGRES_PASSWORD` | dev環境のPostgreSQLパスワード |
| `PROD_POSTGRES_PASSWORD` | prod環境のPostgreSQLパスワード |
| `BASIC_AUTH_DEV_USER` | `dev.mc-werewolf.com` 全体のBasic認証ユーザー名 |
| `BASIC_AUTH_DEV_HASH` | 同上のパスワードのbcryptハッシュ |
| `BASIC_AUTH_ADMIN_USER` | `mc-werewolf.com/admin` のBasic認証ユーザー名 |
| `BASIC_AUTH_ADMIN_HASH` | 同上のパスワードのbcryptハッシュ |

`POSTGRES_USER` / `POSTGRES_DB` は値自体に秘匿性がないため固定値(`werewolf`)とし、Secret化していない。

## ポート構成

| サービス | dev.mc-werewolf.com (ホスト) | mc-werewolf.com (ホスト) | コンテナ内部 |
| --- | --- | --- | --- |
| backend (Go) | 8080 | 8000 | 8000 |
| frontend (Next.js) | 3001 | 3000 | 3000 |
| PostgreSQL | 5433 | 5432 | 5432 |

- コンテナ内部は常に本番用のポート番号を使用する
- dev/prodの違いはdocker composeのホスト側ポートマッピングのみで吸収する

## リバースプロキシ (Caddy)

- 実体: [`deploy/caddy/docker-compose.yml`](../deploy/caddy/docker-compose.yml) + [`deploy/caddy/Caddyfile`](../deploy/caddy/Caddyfile)
- Caddyは dev/prod どちらのスタックにも属さない、**共有の第三のdocker composeスタック**(project名 `ww-caddy`)として稼働する
  - dev/prodそれぞれの app スタック(`ww-dev` / `ww-prod`)が作る docker network(`ww-dev_default` / `ww-prod_default`)の両方に、Caddyコンテナを外部ネットワークとして接続する
  - これにより、Caddy1つで両ドメインへの80/443アクセスを受けつつ、それぞれの環境のbackend/frontendへ到達できる
  - backend/frontendは `deploy/docker-compose.yml` 側で `container_name: ${DEPLOY_ENV_NAME}-backend` / `${DEPLOY_ENV_NAME}-frontend` を明示的に付与しており、Caddyfileはこの固定名(`dev-backend` / `dev-frontend` / `prod-backend` / `prod-frontend`)を参照する
    - (同一エイリアス名を持つ複数ネットワークに接続する場合、docker composeのデフォルトサービス名解決は曖昧になりうるため、明示的なcontainer_nameで一意にしている)
  - **前提条件**: Caddyは起動時に両方の外部networkが存在している必要があるため、初回は dev/prod 両方のアプリスタックを先に1回ずつデプロイしてからでないと `ww-caddy` の起動に失敗する
- ルーティング
  - `dev.mc-werewolf.com` → `/api/*` は `dev-backend:8000`、それ以外は `dev-frontend:3000`
  - `mc-werewolf.com` → `/admin*` は Basic認証つきで `prod-frontend:3000`、`/api/*` は `prod-backend:8000`、それ以外は `prod-frontend:3000`
- Caddyの設定ファイル(`Caddyfile`)も本リポジトリで管理し、変更はコード経由(git管理下)で行う
- TLS証明書の取得・更新はCaddyの自動HTTPS機能に任せる(Let's Encrypt連携が組み込みのため、別途certbot等は不要)。ローカル検証では実DNSが無いため証明書取得までは確認できないが、Caddyfileの構文検証・HTTP→HTTPSリダイレクト・コンテナ間到達性(dev/prod双方のbackend/frontendへの疎通)は確認済み

### Basic認証

| 対象 | スコープ |
| --- | --- |
| `dev.mc-werewolf.com` | サブドメイン全体 |
| `mc-werewolf.com/admin` | `/admin` パスのみ(それ以外の本番ページは認証なしで公開) |

- いずれも Caddyfile の `basic_auth` ディレクティブで保護する
- ユーザー名・パスワードハッシュは環境変数(`{$BASIC_AUTH_DEV_USER}` 等)経由でCaddyfileに埋め込み、実際の値はGitHub Secretsで管理する
- ハッシュの生成: `docker run --rm caddy:2-alpine caddy hash-password --plaintext '<パスワード>'`(bcryptハッシュが出力されるので、そのままSecretへ登録する)

## DBマイグレーション

- ツール: [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate)(`v4`)
- マイグレーションファイルは `backend/internal/migrate/migrations/` にSQLとして配置し、`go:embed` でbackendバイナリに埋め込む
- backend起動時に自動で未適用のマイグレーションを適用する(`migrate.Up()`)。Postgresのadvisory lockを使うため多重起動時も安全
  - これにより、dev/prodいずれもデプロイ(コンテナ起動)のたびに自動でスキーマが最新化される。GitHub Actions側で別途マイグレーション用のステップは不要
  - マイグレーション(または接続)に失敗すると `log.Fatalf` でbackendプロセスごと終了する。この失敗はdocker composeのhealthcheckで検知され、デプロイジョブ自体を失敗させる(後述「堅牢性」参照)。**「毎回安全に実行できるか」は idempotent + advisory lock で担保、「失敗時に前進しないか」は healthcheck + `--wait` で担保**、という2段構え
  - DB接続文字列(`postgres://user:password@host/db`)はbackend内で `net/url.UserPassword` を使い安全に組み立てている(単純な文字列結合だと、生成されたパスワードに `^` `%` `|` 等の記号が含まれた場合にURLとして壊れ、backendが起動時に無限に再起動し続けるバグを実際に踏んだため)
- ローカルでのマイグレーションファイル作成・手動適用/ロールバックは `Makefile` の `migrate-create` / `migrate-up` / `migrate-down` を使う(公式 `migrate/migrate` Dockerイメージ経由。ローカルにmigrate CLIのインストールは不要)

## 堅牢性

### デプロイ失敗の検知(fail-fast)

以前は `docker compose up -d` を実行するだけで、コンテナが起動直後にクラッシュしていてもデプロイジョブは「成功」と表示されていた(実際にbackendのDB接続文字列が壊れて再起動ループしていたのに気づかなかった実例がある)。これを防ぐため:

- `postgres` / `backend` / `frontend` / `caddy` それぞれに `healthcheck` を定義(`deploy/docker-compose.yml`, `deploy/caddy/docker-compose.yml`, ローカル用 `docker-compose.yml` 共通)
  - `postgres`: `pg_isready`
  - `backend`: `GET /api/health` へのwget
  - `frontend`: `GET /` へのwget
  - `caddy`: admin API(`:2019/config/`)へのwget
  - **healthcheckのURLは `127.0.0.1` を使うこと**(`localhost` はIPv6の `::1` に解決されることがあり、Next.jsの `HOSTNAME=0.0.0.0` はIPv4のみのbindのため `localhost` だと接続拒否になる不具合を実際に踏んだ)
- `backend`/`frontend` の `depends_on` は `condition: service_healthy` を使い、前段が本当に健康になってから次を起動する
- デプロイスクリプト側は `docker compose ... up -d --wait --wait-timeout <N>` を使う。`--wait` は全サービスがhealthyになるまで待ち、タイムアウトまたはコンテナが終了した場合は非ゼロ終了する。これによりGitHub Actionsのジョブ自体が失敗として報告される
- ローカルの `make up` も同様に `--wait` 付き(`Makefile`参照)。実際に不正なDBパスワードを与えて `up -d --wait` がexit code 1で失敗することを確認済み

### データ永続化・分離(確認済み)

- PostgreSQLデータは named volume(`ww-dev_postgres_data` / `ww-prod_postgres_data`)に保存され、`docker compose ... up -d` の再実行(redeploy)では消えない(volumeを明示的に消すのは `down -v` のみ。デプロイスクリプトはこれを使わない)
- dev/prodは volume名・コンテナ名・ネットワークすべてが `ww-dev` / `ww-prod` のcompose project名で分離されており、データが混ざることはない
- Caddyの証明書データ(`caddy_data:/data`, `caddy_config:/config`)も同様にnamed volumeで永続化されており、実際に発行された `mc-werewolf.com` / `dev.mc-werewolf.com` のLet's Encrypt証明書が保存されていることを確認済み

### 既知の制約: redeploy時の瞬断

`backend`/`frontend` イメージが変わるredeployでは、docker composeが古いコンテナを停止・削除してから新しいコンテナを起動するため、その間の数秒〜数十秒、Caddyから見て `dev-backend` 等の名前解決が一時的にできなくなる瞬間がある(実際に発生を確認)。現状はこの間502が返るのみで実害は小さいが、ゼロダウンタイムデプロイ(blue/green等)は未実装。

## サーバー外バックアップ

- ワークフロー: [`.github/workflows/backup.yml`](../.github/workflows/backup.yml)
- 毎日(19:00 UTC = 04:00 JST)+ 手動実行(`workflow_dispatch`)で、prod環境のPostgreSQLを `pg_dump -Fc`(custom format。デフォルトでzlib圧縮される。実測で非圧縮の約1/19のサイズになることを確認済み)でダンプする
- 保存先は2箇所(冗長化):
  1. **GitHub Actions artifact**(保持期間30日)
  2. **Google Drive**(個人のDriveに作成したフォルダへアップロード。期限なし)
     - GitHub Actionsのランナー上で `rclone` をインストールし、`~/.config/rclone/rclone.conf` を直接書き込んで設定
     - **認証方式はOAuth(本人のGoogleアカウントで認可した refresh token を含む rclone設定)を使う**。サービスアカウント方式は一度試したが、Googleの仕様上サービスアカウントは自前のストレージ容量を持たず、個人のDrive(共有フォルダ経由でも)へは `storageQuotaExceeded` で書き込めないため不採用とした
     - OAuthのセットアップ(Google Cloud上でのOAuthクライアントID作成 → ローカルで `rclone config` により認可 → 生成された `rclone.conf` の中身をそのままSecretへ)は人手による一度きりの作業が必要
     - 必要なSecrets: `RCLONE_CONFIG`(`~/.config/rclone/rclone.conf` の中身全体。client_id/client_secret/refresh tokenを含む)
     - アップロード先は `gdrive:minecraft/werewolf/server/backups`(マイドライブ配下、中間フォルダはrcloneが自動作成。フォルダIDでの指定はしない方針)
- サーバー自体が失われても、上記いずれかにバックアップが残る
- 手動でのバックアップ取得・復元手順は [`README.md`](../README.md) に記載。実際に本番相当のダンプを取得し、別のPostgreSQLコンテナへ `pg_restore` で復元できることを確認済み
- devのバックアップは現状対象外(必要になれば同様の仕組みを追加)

## ローカル開発

- dev/prodのようなポートの使い分けは行わず、常に本番ポート(backend:8000, frontend:3000, postgres:5432)でホストにバインドする
- ルートの `Makefile` から `make up` を実行することで、上記3サービスがdocker compose経由で起動する
- 詳細は [`docker-compose.yml`](../docker-compose.yml) / [`Makefile`](../Makefile) を参照

## 未確定事項 / TODO

- CIでのテスト実行(現状のワークフローはビルド・デプロイのみで、テストステップは未追加)
- Caddyの証明書データ(`caddy_data`)自体のバックアップ(現状は未バックアップ。失っても再発行は自動で行われるため優先度は低い)
- `mc-werewolf.com/admin` の実体(Next.js内の管理画面ルートか、別アプリか)、Basic認証に加えたアプリ側認証の要否
- `/opt/werewolf` ディレクトリ・Dockerのインストールなど、サーバー初期セットアップの具体的な手順の文書化(実施はしたが、手順書としては未整理)
- redeploy時のゼロダウンタイム化(現状はbackend/frontend入れ替え時に数秒〜数十秒の502が発生しうる)
- Google Driveに溜まり続けるバックアップの世代管理・自動削除(現状は増え続ける一方)
- devデータベースのバックアップ(現状prodのみ)
