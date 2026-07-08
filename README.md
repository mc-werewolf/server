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

[`.github/workflows/backup.yml`](.github/workflows/backup.yml) が毎日 (19:00 UTC / 04:00 JST) prod環境のPostgreSQLを `pg_dump -Fc`(圧縮済み)でダンプし、以下2箇所へアップロードする。手動実行(`workflow_dispatch`)も可能。

1. **GitHub Actions artifact**(保持期間30日、過ぎると自動削除)
2. **Google Drive**(サービスアカウント経由で個人のDriveの共有フォルダへアップロード。期限なしだが、世代管理・自動削除は無いため増え続ける点に注意)

サーバー本体が壊れても上記いずれかにバックアップが残るため、サーバー外バックアップとして機能する。

### 手動バックアップ

```
ssh <DEPLOY_USER>@<DEPLOY_HOST> "docker exec ww-prod-postgres-1 pg_dump -U werewolf -Fc werewolf" > werewolf-prod-backup.dump
```

### 復元手順

1. GitHub Actionsの `Backup Prod Database` ワークフローの実行結果から、対象のartifact(`werewolf-prod-*.dump`)をダウンロードする(または上記の手動バックアップで取得したファイルを使う)
2. 復元先のPostgreSQLコンテナにダンプファイルをコピーする

   ```
   docker cp werewolf-prod-backup.dump ww-prod-postgres-1:/tmp/backup.dump
   ```

3. `pg_restore` で復元する(`--clean --if-exists` で既存オブジェクトを削除してから復元)

   ```
   docker exec ww-prod-postgres-1 pg_restore -U werewolf -d werewolf --clean --if-exists /tmp/backup.dump
   ```

4. 復元後、`https://mc-werewolf.com/api/health/db` などで疎通を確認する

**注意**: 本番環境に対して復元コマンドを実行すると、既存データが上書きされる。実行前に必ず現在のデータもバックアップしておくこと。
