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
  - 具体的には docker compose の projectを dev/prod で分ける(例: `docker compose -p werewolf-dev ...` / `docker compose -p werewolf-prod ...`)ことで、コンテナ名・ネットワーク・named volume(PostgreSQLのデータ含む)を自動的に分離する
  - dev/prodそれぞれ専用の環境変数ファイル(DB名・ユーザー・パスワード等)を用いる。値そのものを同じにしない
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

`build-and-push` / prodデプロイ前backup / app deploy / Caddy deploy の一連の流れは、[`kairo-js/github-workflows`](https://github.com/kairo-js/github-workflows) 側の再利用可能ワークフロー(`web-app-release.yml`)に切り出されている。内部では `docker-build-push.yml` / `postgres-backup.yml` / `app-deploy.yml` / `caddy-snippet-deploy.yml` を組み合わせている。kairo-server / blog-app など、同一(または別)ホストに載る他サービスも同じワークフローを呼び出す想定で、werewolf-server 側の `deploy.yml` は各サービス固有の値(`app-name` / `image-prefix` / DB名 / Caddy snippetテンプレート)を渡すだけの薄いラッパーになっている。

**Caddy本体を管理する別リポジトリは存在しない。** `caddy-snippet-deploy.yml` が完全に自己完結しており、共有Caddy(TLS終端・80/443待受、コンテナ名 `proxy-caddy`)のcompose定義をワークフロー内に直接持っていて、呼ばれるたびに「本体が存在し最新であることを保証」→「自分のsnippetを配置してreload」を行う。werewolf-server が持つのは自分のルーティング設定(`deploy/caddy/service.caddy`)だけ。詳細は後述「リバースプロキシ (Caddy)」を参照。

1. **build-and-push ジョブ**
   - backend / frontend それぞれの Docker イメージをビルドし、Docker Hub の `<DOCKERHUB_USERNAME>/mc-werewolf-backend` / `<DOCKERHUB_USERNAME>/mc-werewolf-frontend` にpush
   - イメージタグ: `main` へのpush時は `dev`、`v*.*.*` タグ作成時はそのタグ名(例 `v1.0.0`)をそのままタグとして使用
2. **pre-prod-backup ジョブ**(tag push時のみ、実体は `kairo-js/github-workflows` の `postgres-backup.yml` 呼び出し)
   - prod デプロイで既存DBを書き換える前に、現在の prod PostgreSQL を `pg_dump -Fc` で取得し、GitHub Actions artifact と Google Drive へ保存する
   - 初回 prod デプロイでは `/opt/werewolf/prod/docker-compose.yml` がまだ存在しないため、`skip-if-missing: true` により「バックアップ対象なし」として成功扱いで通過する
   - compose定義が存在するのに `pg_dump` に失敗した場合は skip せず、prod デプロイ自体を止める
3. **deploy ジョブ**(`needs: build-and-push`、prodでは `pre-prod-backup` 成功後。実体は `kairo-js/github-workflows` の `app-deploy.yml` 呼び出し)
   - デプロイ専用の [`deploy/docker-compose.yml`](../deploy/docker-compose.yml)(`build:` ではなく `image:` 参照。ローカル開発用のルート `docker-compose.yml` とは別ファイル)を SCP でサーバー上の `/opt/<app-name>/<env>/` (werewolfでは `/opt/werewolf/<env>/`) に配置
   - SSH接続し、`docker network create proxy`(既にあれば何もしない、冪等)を実行した上で、環境ごとの `.env`(アプリ名・イメージprefix・DB認証情報・イメージタグ等)をSecrets/`with:`入力から生成し、`docker compose -p <app-name>-<env> --env-file .env pull && up -d --wait --wait-timeout 90` を実行
   - `<app-name>` は `werewolf`(呼び出し側の `deploy.yml` で指定)、`<env>` は `dev` / `prod`。docker composeの **project名にapp名を含めて分ける**ことで、複数サービスが同一ホストに同居してもコンテナ・ネットワーク・named volume(DBデータ含む)が衝突せず分離される
   - `--wait` は各サービスの healthcheck が通るまでジョブを待たせ、タイムアウト・失敗時は非ゼロ終了でジョブごと失敗させる(後述「堅牢性」参照)
4. **deploy-caddy ジョブ**(`needs: deploy`、実体は `kairo-js/github-workflows` の `caddy-snippet-deploy.yml` 呼び出し)
   - まず `deploy-proxy-body` サブジョブが、`PROXY_HOST` 上に `proxy` network(存在しなければ作成)と `proxy-caddy` コンテナ(存在しなければ起動、あれば `pull && up -d --wait` で最新化)を保証する。compose定義・`Caddyfile` はこのワークフロー内に直接書かれており、どこか別のリポジトリを参照しない
   - 次に `deploy-snippet` サブジョブが、werewolf-server の [`deploy/caddy/service.caddy`](../deploy/caddy/service.caddy) を checkout し、`envsubst` で `$BASIC_AUTH_DEV_USER` 等のplaceholderをSecretsの実値に置換した上で `/opt/proxy/conf.d/werewolf.caddy` へ配置し、`docker exec proxy-caddy caddy fmt --overwrite ...` → `docker exec proxy-caddy caddy reload ...` を実行する
   - dev/prodどちらのpushでも毎回実行される(Caddyは環境共有のため)
   - **前提条件**: `proxy-caddy` はwerewolf-dev/werewolf-prod両方のアプリコンテナが参加する `proxy` networkに自身も参加するが、その`proxy` networkの存在自体は `deploy` ジョブ側(`docker network create proxy`)でも `deploy-proxy-body` 側でも作成されるため、**dev/prodどちらを先にデプロイしても初回から成功する**(後述「リバースプロキシ (Caddy)」参照)

### 必要なGitHub Secrets / Variables

| Secret | 用途 |
| --- | --- |
| `DOCKERHUB_USERNAME` | Docker Hubログイン、イメージの名前空間 |
| `DOCKERHUB_TOKEN` | Docker Hubログイン用アクセストークン |
| `DEPLOY_HOST` | デプロイ先サーバーのホスト名/IP |
| `DEPLOY_USER` | デプロイ先サーバーのSSHユーザー |
| `DEPLOY_SSH_KEY` | デプロイ先サーバーへのSSH秘密鍵 |
| `PROXY_HOST` | Caddy(proxy)ホストのホスト名/IP。同居構成では `DEPLOY_HOST` と同値 |
| `PROXY_USER` | 同上のSSHユーザー。同居構成では `DEPLOY_USER` と同値 |
| `PROXY_SSH_KEY` | 同上へのSSH秘密鍵。同居構成では `DEPLOY_SSH_KEY` と同値 |
| `DEV_POSTGRES_PASSWORD` | dev環境のPostgreSQLパスワード |
| `PROD_POSTGRES_PASSWORD` | prod環境のPostgreSQLパスワード |
| `BASIC_AUTH_DEV_USER` | `dev.mc-werewolf.com` 全体のBasic認証ユーザー名 |
| `BASIC_AUTH_DEV_HASH` | 同上のパスワードのbcryptハッシュ |
| `BASIC_AUTH_ADMIN_USER` | `mc-werewolf.com/admin` のBasic認証ユーザー名 |
| `BASIC_AUTH_ADMIN_HASH` | 同上のパスワードのbcryptハッシュ |
| `RCLONE_CONFIG` | Google Drive backup 用の rclone 設定全体 |
| `PROD_GDRIVE_DESTINATION` | prod backup の Google Drive 保存先 |
| `DEV_GDRIVE_DESTINATION` | dev backup の Google Drive 保存先 |

| Variable(非機密) | 用途 |
| --- | --- |
| `ACME_EMAIL` | 共有Caddyインスタンスの Let's Encrypt ACMEアカウントに使うメールアドレス。同じ `PROXY_HOST` を共有する全サービスで同じ値にすること |

`POSTGRES_USER` / `POSTGRES_DB` は値自体に秘匿性がないため固定値(`werewolf`)とし、Secret化していない。werewolf-serverのBasic認証secretsは `caddy-snippet-deploy.yml` 側には渡らない(snippetレンダリング時点で実値に変換済みのものを文字列として渡すだけなので、ワークフロー自体はsecret名を一切知らない)。

## ネットワーク構成

| サービス | ホスト公開 | コンテナ内部 |
| --- | --- | --- |
| backend (Go) | なし(Caddyから `proxy` network 経由) | 8000 |
| frontend (Next.js) | なし(Caddyから `proxy` network 経由) | 3000 |
| PostgreSQL | なし(アプリ専用network内のみ) | 5432 |

- コンテナ内部は常に本番用のポート番号を使用する
- dev/prodおよび別サービスの分離は compose project名とコンテナ名で行い、ホスト側ポートの取り合いを発生させない

## リバースプロキシ (Caddy)

Caddy本体を管理する別リポジトリは無い。**共有Caddy(TLS終端・80/443待受、コンテナ名 `proxy-caddy`)は `kairo-js/github-workflows` の `caddy-snippet-deploy.yml` が完全に自己完結で管理する。** werewolf-server が持つのは自分のルーティング設定のsnippet([`deploy/caddy/service.caddy`](../deploy/caddy/service.caddy))だけ。

- **Caddy本体は特定のアプリ名を一切知らない。** `proxy-caddy` の compose定義は「`proxy` という汎用networkに参加する」としか書かれておらず、werewolf/kairo/blogのような個別の名前やnetworkはCaddy側のどのファイルにも登場しない。`Caddyfile` は `import /etc/caddy/conf.d/*.caddy` のみの固定内容で、サービスが増えても変更不要
- **`proxy` network がすべてを繋ぐ唯一の共有点。**
  - `app-deploy.yml`(アプリ本体のデプロイ)と `caddy-snippet-deploy.yml`(Caddy側)の両方が、自分の実行時に `docker network create proxy`(既にあれば何もしない)を行う。**どちらが先にデプロイされても必ず成功する**ため、dev/prodデプロイの順序に関する制約は無い
  - werewolf-serverの [`deploy/docker-compose.yml`](../deploy/docker-compose.yml) は、backend/frontendを `default`(postgres接続用の自分専用network)に加えて `proxy`(外部network)にも参加させている。postgresは `proxy` に参加させていない(Caddyから直接到達する必要が無いため)
  - `kairo-server` や `blog-app` など将来増えるサービスも、同じ `proxy` networkに参加しさえすれば、Caddy側・他サービス側は何も変更しなくてよい
- werewolf-server 側の役割
  - [`deploy/caddy/service.caddy`](../deploy/caddy/service.caddy) に、werewolf-serverが管理するドメインのCaddy設定(サイトブロックそのもの)を書く。`import`で展開される前提のsnippetなので、書き方自体は従来のCaddyfileと同じ(トップレベルにサイトアドレスブロックを並べるだけ)
  - backend/frontendは `deploy/docker-compose.yml` 側で `container_name: ${APP_NAME}-${DEPLOY_ENV_NAME}-backend` / `-frontend` を明示的に付与しており、snippetはこの固定名(`werewolf-dev-backend` / `werewolf-dev-frontend` / `werewolf-prod-backend` / `werewolf-prod-frontend`)を参照する
    - (同一エイリアス名を持つ複数ネットワークに接続する場合、docker composeのデフォルトサービス名解決は曖昧になりうるため、明示的なcontainer_nameで一意にしている)
  - `deploy.yml` の `deploy-caddy` ジョブが `kairo-js/github-workflows` の `caddy-snippet-deploy.yml` へ snippetテンプレートのパスとBasic認証Secretsを渡し、`proxy-caddy` 本体の保証 → snippetレンダリング → `/opt/proxy/conf.d/werewolf.caddy` への配置 → `caddy reload` を行う(前述「GitHub Actions (CI/CD)」参照)
- サーバーの分離・同居について: `PROXY_HOST`/`PROXY_USER`/`PROXY_SSH_KEY` は(`DEPLOY_HOST`等と同様に)werewolf-server自身のSecretsとして持つ。kairo-serverが将来別サーバーに立つ場合、kairo-server側は自分のSecretsで別の `PROXY_HOST` を指すだけでよく、werewolf-server側・共通ワークフロー側のどちらも変更不要。同居させたい場合は単に同じ `PROXY_HOST` を指せばよい
- ルーティング(snippetの中身)
  - `dev.mc-werewolf.com` → `/api/*` は `werewolf-dev-backend:8000`、それ以外は `werewolf-dev-frontend:3000`
  - `mc-werewolf.com` → `/admin*` は Basic認証つきで `werewolf-prod-frontend:3000`、`/api/*` は `werewolf-prod-backend:8000`、それ以外は `werewolf-prod-frontend:3000`
- TLS証明書の取得・更新はCaddyの自動HTTPS機能に任せる(Let's Encrypt連携が組み込みのため、別途certbot等は不要)。証明書データは `proxy-caddy` のnamed volume(`caddy_data`/`caddy_config`、project名 `proxy`)に保存される

### Basic認証

| 対象 | スコープ |
| --- | --- |
| `dev.mc-werewolf.com` | サブドメイン全体 |
| `mc-werewolf.com/admin` | `/admin` パスのみ(それ以外の本番ページは認証なしで公開) |

- いずれも snippet の `basic_auth` ディレクティブで保護する
- ユーザー名・パスワードハッシュは werewolf-server の GitHub Secrets で管理し、`caddy-snippet-deploy.yml` の `deploy-snippet` ジョブが `envsubst` で `$BASIC_AUTH_DEV_USER` 等のplaceholderを実値に展開してから配置する(Caddyランタイムの環境変数展開機能は使わない。`proxy-caddy` コンテナはこれらのsecretを一切知らない)
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

- `postgres` / `backend` / `frontend` それぞれに `healthcheck` を定義(`deploy/docker-compose.yml`, ローカル用 `docker-compose.yml` 共通)
  - `postgres`: `pg_isready`
  - `backend`: `GET /api/health` へのwget
  - `frontend`: `GET /` へのwget
  - Caddy(`proxy-caddy`)のhealthcheck(admin API `:2019/config/`)は `caddy-snippet-deploy.yml` 内のcompose定義で行う
  - **healthcheckのURLは `127.0.0.1` を使うこと**(`localhost` はIPv6の `::1` に解決されることがあり、Next.jsの `HOSTNAME=0.0.0.0` はIPv4のみのbindのため `localhost` だと接続拒否になる不具合を実際に踏んだ)
- `backend`/`frontend` の `depends_on` は `condition: service_healthy` を使い、前段が本当に健康になってから次を起動する
- デプロイスクリプト側は `docker compose ... up -d --wait --wait-timeout <N>` を使う。`--wait` は全サービスがhealthyになるまで待ち、タイムアウトまたはコンテナが終了した場合は非ゼロ終了する。これによりGitHub Actionsのジョブ自体が失敗として報告される
- ローカルの `make up` も同様に `--wait` 付き(`Makefile`参照)。実際に不正なDBパスワードを与えて `up -d --wait` がexit code 1で失敗することを確認済み

### データ永続化・分離(確認済み)

- PostgreSQLデータは named volume(`werewolf-dev_postgres_data` / `werewolf-prod_postgres_data`)に保存され、`docker compose ... up -d` の再実行(redeploy)では消えない(volumeを明示的に消すのは `down -v` のみ。デプロイスクリプトはこれを使わない)
- dev/prodは volume名・コンテナ名・ネットワークすべてが `werewolf-dev` / `werewolf-prod` のcompose project名で分離されており、データが混ざることはない
- Caddyの証明書データ(`caddy_data:/data`, `caddy_config:/config`)は `proxy-caddy` 自身のnamed volume(project名 `proxy`)で永続化される

### 初回セットアップ手順(サーバー初期化済み・データ移行不要)

サーバー自体を作り直したため、旧 `ww-*` 環境やそのDB/証明書データの移行は不要(初期段階のためデータの重要性も無い)。ゼロから以下の順で立ち上げる:

1. サーバーに Docker / Docker Compose plugin をインストールし、`DEPLOY_SSH_KEY` / `PROXY_SSH_KEY` に対応する公開鍵を `DEPLOY_USER` / `PROXY_USER` の `authorized_keys` に登録する(これはどのGitHub Actionsワークフローも自動化しない、手動の前提作業)
2. werewolf-serverリポジトリに以下のSecrets/Variablesを設定する: `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` / `DEPLOY_HOST` / `DEPLOY_USER` / `DEPLOY_SSH_KEY` / `PROXY_HOST` / `PROXY_USER` / `PROXY_SSH_KEY` / `DEV_POSTGRES_PASSWORD` / `PROD_POSTGRES_PASSWORD` / `BASIC_AUTH_DEV_USER` / `BASIC_AUTH_DEV_HASH` / `BASIC_AUTH_ADMIN_USER` / `BASIC_AUTH_ADMIN_HASH`(Secrets)、`ACME_EMAIL`(Variable)
3. `main` へpush(dev)、次に `v*.*.*` タグ作成(prod)の順にデプロイする。どちらも `deploy`(アプリ本体、`proxy` networkを作成)→ `deploy-caddy`(`proxy-caddy` 本体を保証 → snippet配置・reload)まで一気通貫で成功するはず(前述の通り `proxy` network はどちらが先でも自己解決するため、順序に神経質になる必要はない)
4. `https://dev.mc-werewolf.com` / `https://mc-werewolf.com` への疎通・証明書取得を確認する

### 既知の制約: redeploy時の瞬断

`backend`/`frontend` イメージが変わるredeployでは、docker composeが古いコンテナを停止・削除してから新しいコンテナを起動するため、その間の数秒〜数十秒、Caddyから見て `werewolf-dev-backend` 等の名前解決が一時的にできなくなる瞬間がある(実際に発生を確認)。現状はこの間502が返るのみで実害は小さいが、ゼロダウンタイムデプロイ(blue/green等)は未実装。

## サーバー外バックアップ

- ワークフロー: [`.github/workflows/backup.yml`](../.github/workflows/backup.yml)
- 毎日(19:00 UTC = 04:00 JST)で、prod環境のPostgreSQLを `pg_dump -Fc`(custom format。デフォルトでzlib圧縮される。実測で非圧縮の約1/19のサイズになることを確認済み)でダンプする。毎日の dump は prod 保存先配下の `daily/` に保存する。手動実行(`workflow_dispatch`)では dev/prod を選択でき、各環境の `manual/` に保存する。加えて、tag push による prod デプロイ前にも prod 保存先配下の `versions/` へ `werewolf-prod-before-<tag>-*` としてダンプする
- 保存先は2箇所(冗長化):
  1. **GitHub Actions artifact**(保持期間30日)
  2. **Google Drive**(個人のDriveに作成したフォルダへアップロード)
     - GitHub Actionsのランナー上で `rclone` をインストールし、`~/.config/rclone/rclone.conf` を直接書き込んで設定
     - **認証方式はOAuth(本人のGoogleアカウントで認可した refresh token を含む rclone設定)を使う**。サービスアカウント方式は一度試したが、Googleの仕様上サービスアカウントは自前のストレージ容量を持たず、個人のDrive(共有フォルダ経由でも)へは `storageQuotaExceeded` で書き込めないため不採用とした
     - OAuthのセットアップ(Google Cloud上でのOAuthクライアントID作成 → ローカルで `rclone config` により認可 → 生成された `rclone.conf` の中身をそのままSecretへ)は人手による一度きりの作業が必要
     - 必要なSecrets: `RCLONE_CONFIG`(`~/.config/rclone/rclone.conf` の中身全体。client_id/client_secret/refresh tokenを含む)、`PROD_GDRIVE_DESTINATION`(例: `minecraft/werewolf/server/prod/backups`)、`DEV_GDRIVE_DESTINATION`(例: `minecraft/werewolf/server/dev/backups`)
     - アップロード先は毎日の prod backup なら `gdrive:${PROD_GDRIVE_DESTINATION}/daily`、prod deploy 前 backup なら `gdrive:${PROD_GDRIVE_DESTINATION}/versions`、prod 手動 backup なら `gdrive:${PROD_GDRIVE_DESTINATION}/manual`、dev 手動 backup なら `gdrive:${DEV_GDRIVE_DESTINATION}/manual`(マイドライブ配下、中間フォルダはrcloneが自動作成。フォルダIDでの指定はしない方針)
- サーバー自体が失われても、上記いずれかにバックアップが残る
- 手動でのバックアップ取得・復元手順は [`README.md`](../README.md) に記載。実際に本番相当のダンプを取得し、別のPostgreSQLコンテナへ `pg_restore` で復元できることを確認済み
- devのバックアップは手動実行のみ対象とし、`gdrive:${DEV_GDRIVE_DESTINATION}/manual` に保存する

## ローカル開発

- dev/prodのようなポートの使い分けは行わず、ローカルでは常に本番ポート(backend:8000, frontend:3000, postgres:5432)でホストにバインドする
- ルートの `Makefile` から `make up` を実行することで、上記3サービスがdocker compose経由で起動する
- 詳細は [`docker-compose.yml`](../docker-compose.yml) / [`Makefile`](../Makefile) を参照

## 未確定事項 / TODO

- CIでのテスト実行(現状のワークフローはビルド・デプロイのみで、テストステップは未追加)
- Caddyの証明書データ(`caddy_data`)自体のバックアップ(現状は未バックアップ。失っても再発行は自動で行われるため優先度は低い)
- `mc-werewolf.com/admin` の実体(Next.js内の管理画面ルートか、別アプリか)、Basic認証に加えたアプリ側認証の要否
- `/opt/werewolf` ディレクトリ・Dockerのインストールなど、サーバー初期セットアップの具体的な手順の文書化(実施はしたが、手順書としては未整理)
- redeploy時のゼロダウンタイム化(現状はbackend/frontend入れ替え時に数秒〜数十秒の502が発生しうる)
- Google Driveに溜まり続けるバックアップの世代管理・自動削除(現状は増え続ける一方)
- 上記「初回セットアップ手順」の実施(Secrets/Variablesの設定、dev→prodの順デプロイ、疎通確認)
