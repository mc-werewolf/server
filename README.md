# Werewolf Server

マイクラ人狼の公式Linuxサーバー(backend + frontend)。詳細は [`specs/`](./specs) を参照。

## ローカル開発

```
make up
```

- backend: http://localhost:8000 (`/api/health`, `/api/health/db`, dev環境では `/api/swagger/index.html`)
- frontend: http://localhost:3000

## バックアップ・復元

### 自動バックアップ

[`.github/workflows/backup.yml`](.github/workflows/backup.yml) が毎日 (19:00 UTC / 04:00 JST) prod環境のPostgreSQLを `pg_dump -Fc`(圧縮済み)でダンプし、以下2箇所へアップロードする。毎日の dump は Google Drive の prod 保存先配下の `daily/` に保存し、30日を超えた `.dump` は削除する。手動実行(`workflow_dispatch`)では dev/prod を選択でき、各環境の `manual/` に保存する。tag push による prod デプロイ前にも、prod 保存先の `versions/` へ `werewolf-prod-before-<tag>-*` として自動バックアップする。

1. **GitHub Actions artifact**(保持期間30日、過ぎると自動削除)
2. **Google Drive**(OAuthで認可した個人アカウントの `マイドライブ/<PROD_GDRIVE_DESTINATION>/daily/`、`manual/`、`versions/`、または `マイドライブ/<DEV_GDRIVE_DESTINATION>/manual/` へアップロード。フォルダIDではなく rclone のパス指定を使う)
   - サービスアカウントは個人のDriveに書き込めない(`storageQuotaExceeded`)ため、`rclone`のOAuth認可(本人のGoogleアカウントで一度だけ許可し、以後はrefresh tokenで自動更新)を使っている

サーバー本体が壊れても上記いずれかにバックアップが残るため、サーバー外バックアップとして機能する。

### 手動バックアップ

```
ssh <DEPLOY_USER>@<DEPLOY_HOST> "cd /opt/werewolf/prod && docker compose -p werewolf-prod exec -T postgres pg_dump -U werewolf -Fc werewolf" > werewolf-prod-backup.dump
```

### 復元手順

1. GitHub Actionsの `Backup Prod Database` ワークフローの実行結果から、対象のartifact(`werewolf-prod-*.dump`)をダウンロードする(または上記の手動バックアップで取得したファイルを使う)
2. 復元先のPostgreSQLコンテナにダンプファイルをコピーする

   ```
   cd /opt/werewolf/prod
   docker compose -p werewolf-prod cp werewolf-prod-backup.dump postgres:/tmp/backup.dump
   ```

3. `pg_restore` で復元する(`--clean --if-exists` で既存オブジェクトを削除してから復元)

   ```
   cd /opt/werewolf/prod
   docker compose -p werewolf-prod exec -T postgres pg_restore -U werewolf -d werewolf --clean --if-exists /tmp/backup.dump
   ```

4. 復元後、`https://mc-werewolf.com/api/health/db` などで疎通を確認する

**注意**: 本番環境に対して復元コマンドを実行すると、既存データが上書きされる。実行前に必ず現在のデータもバックアップしておくこと。

### サーバーからの撤退

GitHub Actions の `Undeploy` workflow を手動実行する。`target=all` は Caddy snippet を削除して dev/prod の stack を停止する。`target=dev` / `target=prod` は該当 stack のみ停止し、Caddy snippet は残す。`remove-volumes=true` は PostgreSQL volume も削除するため、移行完了後など明確に不要な場合だけ使う。

### Appサーバー切り出し準備

Caddy upstream は `DEV_BACKEND_UPSTREAM` / `DEV_FRONTEND_UPSTREAM` / `PROD_BACKEND_UPSTREAM` / `PROD_FRONTEND_UPSTREAM` の Repository Variables で上書きできる。未設定なら従来通り `werewolf-dev-backend:8000` など同一Docker network上のcontainer名を使う。werewolf appだけ別サーバーへ移す場合は、`*_PORT` でappホスト側の公開portを設定し、対象の upstream をCaddyから見た `<新appサーバーのIPまたはDNS名>:<port>` に設定する。

`Prepare Next App Host` workflow は `NEXT_DEPLOY_HOST` / `NEXT_DEPLOY_USER` / `NEXT_DEPLOY_SSH_KEY` を使って新appサーバーへdeployし、必要なら現在の `DEPLOY_HOST` からDB dumpを取得して復元する。成功後に GitHub UI で `DEPLOY_*` と `*_UPSTREAM` / `*_PORT` Variables を切り替える。
