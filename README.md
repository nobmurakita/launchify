# Launchify

CLIツールをmacOS LaunchAgentに変換するTUIアプリケーション。

対話的なフォームで設定項目を入力するだけで、plistの生成からサービス登録まで一括で行える。

## インストール

```bash
go install github.com/nobmurakita/launchify@latest
```

または、リポジトリをクローンしてビルド:

```bash
git clone https://github.com/nobmurakita/launchify.git
cd launchify
go build
```

## 使い方

```bash
launchify
```

TUIフォームが起動し、以下の項目を順に入力する:

1. **基本設定** — Program（実行コマンド+引数）、Label（識別子）、WorkingDirectory
2. **環境変数** — `KEY=VALUE` 形式で指定（任意）
3. **実行設定** — ProcessType、RunAtLoad、KeepAlive、スケジュール（Interval / Calendar）
4. **ログ出力** — stdout/stderrそれぞれのファイルパス（初期値: `~/Library/Logs/<Label>.log`）

入力後、生成されるplistをプレビュー表示。確認してロードすると `~/Library/LaunchAgents/` に配置され、`launchctl load` で即座に登録される。

## 機能

- **パス自動解決** — コマンド名だけの入力でフルパスに変換
- **引数のクォート対応** — `command --msg "hello world"` のように空白を含む引数も正しく分割
- **スケジュール設定** — 秒間隔（StartInterval）またはカレンダー指定（分/時/日/月/曜日）
- **KeepAlive** — 常時再起動 / 異常終了時のみ再起動 / なし の3択
- **ProcessType** — Background / Standard / Interactive の3種から選択
- **ログファイル出力** — stdout/stderrを個別に指定パスへ出力。初期値は `~/Library/Logs/<Label>.log`
- **既存サービスの安全な上書き** — 同名plistが存在する場合は確認の上、稼働中サービスを停止してから上書き

## 動作環境

- macOS
- Go 1.25 以上
