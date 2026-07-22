# Werewolf Server 概要

## 目的

Windows専用ランチャーアプリ「bds-launcher」の前提となる、公式Linuxサーバー。
「マイクラ人狼」のプレイに必要な各種データ・情報を管理し、bds-launcher からの問い合わせに応答する。

## 位置づけ

- `bds-launcher` リポジトリ側の仕様(overview / 起動フロー)で「公式Linuxサーバー(未構築)」として言及されている外部システムの実体
- 本リポジトリはそのバックエンドを構築する

## 主な責務

bds-launcher の overview には以下2点が前提として書かれている。

1. アドオン等のバージョン管理・配信
2. プレイヤーのRankなどのアカウント管理

これに加えて、起動フローの記述から以下も本サーバーの責務と考えられる。

3. 専用ワールドデータの配布

さらに、情報公開・アプリ配布のためのWebサイトも本リポジトリのスコープに含める。

4. Webサイト(mc-werewolf.com)による情報公開・アプリダウンロード配布

### 1. アドオンのバージョン管理・配信情報の提供

- 現在の対象アドオン: `kairo`, `kairo-database`
- `game-manager`, `vanillapack`, `additional-roles-1`はRegistry公開後に追加する
- 各アドオンのバージョンと実体はKairo Registryで管理・配布する
- 本サーバーはランチャー構成APIとして、人狼サーバーに必要なアドオンIDとKairo最新版APIの参照先を提供する
  - `GET /api/launcher/v1/config`

  - bds-launcherは構成APIを取得し、各`latestVersionUrl`からバージョン、manifest、ファイルサイズ、SHA-256、ダウンロードURLを取得する
  - ローカル版と比較して更新が必要なアドオンだけをKairoからダウンロードする
  - GitHub Releases同期APIは移行期間中の互換用とし、新しいランチャーフローでは利用しない

### 公開ワールドネットワーク

- ランチャーは`POST /api/network/v1/servers`でワールドを登録し、返された秘密トークンをその起動中だけ保持する
- `PUT /api/network/v1/servers/{id}/heartbeat`を90秒以内の間隔で送り、人数・状態・直接接続または中継接続先を更新する
- `GET /api/network/v1/servers`はleaseが有効なオンラインワールドだけを公開する
- 正常終了時は`DELETE /api/network/v1/servers/{id}`を呼び、異常終了時もlease切れで一覧から消える
- 将来のKairoゲーム内UIはこの一覧を読み、`@minecraft/server-admin`の転送APIで別ワールドへ参加させる
- UPnPで直接公開できないランチャーは認証済みWebSocketを`GET /api/network/v1/servers/{id}/relay`へ接続する
- 中央サーバーはUDP `20000-20099`からワールドごとのポートを割り当て、WebSocketとの間でBedrock UDPデータグラムを双方向中継する

### 2. 専用ワールドデータの配布

- マイクラ人狼専用ワールドのテンプレートデータを保持する
- bds-launcher はローカルに専用ワールドが無い場合、本サーバーから取得する
- ローカルに既にある場合の差分更新・同期方式は未定

### 3. プレイヤーアカウント・Rank管理

- プレイヤーのRankなどのアカウント情報を管理する
- 想定利用者:
  - bds-launcher(表示・参照用)
  - BDS上で稼働するアドオン `kairo` / `kairo-database`(ゲーム結果の記録・反映用)

### 4. Webサイト(mc-werewolf.com)による情報公開・アプリダウンロード配布

- ドメイン: `mc-werewolf.com`
- 一般ユーザー向けに、以下を想定
  - マイクラ人狼に関する情報の閲覧(ルール、アドオン情報など)
  - `bds-launcher` アプリのダウンロード配布
- バックエンドAPI(1〜3の機能)と同じデータを参照する想定(例: 最新バージョン情報の表示)

## 想定される利用者(クライアント)

| クライアント | 用途 |
| --- | --- |
| bds-launcher | 起動時の更新確認(アドオンバージョン照会)、専用ワールドデータ取得 |
| kairo / kairo-database (BDSアドオン) | プレイ中のRank更新・ゲーム結果の記録 |
| 一般ユーザー(ブラウザ) | mc-werewolf.com での情報閲覧・アプリダウンロード |

## 非スコープ

- BDS本体(統合版サーバー)のバージョン管理・配布 → Microsoft公式サーバーから直接取得するため対象外
- Windows側のUI/UX、ランチャー本体の実装 → `bds-launcher` リポジトリのスコープ

## 技術スタック

- バックエンドAPI: Go
- データベース: PostgreSQL(接続はGoの `jackc/pgx/v5` を使用)
- フロントエンド(mc-werewolf.com): Next.js
- リポジトリ構成: モノレポ(backend / frontend を本リポジトリで一括管理)

デプロイ・環境構成の詳細は [`deployment.md`](./deployment.md) を参照。

## API

- ベースパス: `mc-werewolf.com/api/`(backendコンテナ自体が `/api` プレフィックス配下でルーティングする)
- Swagger UI: `dev.mc-werewolf.com/api/swagger`(dev環境のみ有効。`APP_ENV=dev` のときのみ backend がルートを登録する)
- 実装: 標準 `net/http`(Go 1.25の `http.ServeMux` メソッドルーティング)+ `swaggo/swag` によるOpenAPI生成
- 現状のエンドポイント:
  - `GET /api/health` — プロセスの疎通確認用
  - `GET /api/health/db` — PostgreSQLへの接続確認用(`pgxpool.Pool.Ping`。接続不可時は503を返す)
- 詳細は [`backend/`](../backend) を参照

## 未確定事項 / TODO

- API仕様の拡充(エンドポイント一覧、リクエスト/レスポンス形式、認証方式)
- DBスキーマ設計(マイグレーションツールは `golang-migrate` に決定済み。詳細は [`deployment.md`](./deployment.md) 参照)
- Kairo manifestの依存バージョン制約をランチャー側で解決する具体的なアルゴリズム
- 専用ワールドデータの差分更新・同期方式(サーバー側更新をローカルにどう反映するか)
- プレイヤーの識別・認証方式(Xbox Live/Microsoftアカウント連携の有無)
- Rankの算出ロジック・データモデル
- kairo / kairo-database アドオンから本サーバーへの通信仕様(BDSプロセス内からの外部通信経路・認証)
- ダウンロード配布するアプリ本体(bds-launcherのビルド成果物)のホスティング方法(自前ストレージ / GitHub Releases流用など)
- 可用性要件(本サーバーがダウンした場合、bds-launcher 側はオフライン起動を許容するか)
