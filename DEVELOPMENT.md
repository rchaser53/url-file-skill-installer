# Development Guide

この文書は `codex-skills-installer` の開発者向け情報をまとめたものです。利用者向けの使い方は [README.md](./README.md) を参照してください。

## 前提条件

- Go 1.24 以上
- `git` コマンドが利用可能であること

## ローカル開発

依存関係は Go Modules で管理されています。通常は次のコマンドで動作確認できます。

```bash
go test ./...
go vet ./...
```

CLI をローカルで試す場合は次を実行します。

```bash
go run .
```

引数やヘルプ表示を確認する場合:

```bash
go run . --help
```

## リポジトリの主なファイル

- `main.go`: CLI エントリーポイント
- `installer.go`: skill の配置処理
- `config.go`: YAML 設定の読み取り
- `git_source.go`: Git URL の解釈と取得元処理
- `main_test.go`: 主要なテスト

## リリース

タグ push をトリガーに GitHub Actions が検証、ビルド、Release 公開を行います。ワークフロー定義は `.github/workflows/release.yml` にあります。

検証ジョブでは次を実行します。

- `go test ./...`
- `go vet ./...`

例えば次のようにタグを作成して push すると、Release が自動作成されます。

```bash
git tag v1.0.0
git push origin v1.0.0
```

添付される成果物は次の形式です。

- `codex-skills-installer_<tag>_linux_amd64.tar.gz`
- `codex-skills-installer_<tag>_linux_arm64.tar.gz`
- `codex-skills-installer_<tag>_darwin_amd64.tar.gz`
- `codex-skills-installer_<tag>_darwin_arm64.tar.gz`
- `codex-skills-installer_<tag>_windows_amd64.zip`
- `codex-skills-installer_<tag>_windows_arm64.zip`
