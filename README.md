# codex-skills-installer

YAML ファイルに並べた Git リポジトリ URL から、Codex skill を `.codex/skills` 配下へまとめてインストールするための Go CLI です。

このリポジトリには次のものが含まれます。

- `main.go`: URL 一覧ファイルを読み取り、skill をインストールする CLI 本体
- `go.mod`: Go モジュール定義
- `go.sum`: 依存関係のロックファイル

## できること

- YAML の `targets` 配列から skill リポジトリ URL を読み取る
- `outputDir` でインストール先ディレクトリを指定できる
- 各 target の `name` でインストール先ディレクトリ名を指定できる
- 各 target の `version` で branch、tag、commit hash を指定できる
- GitHub URL と一般的な Git リポジトリ URL に対応する
- GitHub の tree URL で指定したディレクトリ配下をそのままコピーできる
- 対象リポジトリ内の `SKILL.md` を検出して `.codex/skills` に配置する

## 前提条件

- Go 1.24 以上
- `git` コマンドが利用可能であること

## クイックスタート

プロジェクトルートに `codex-skills.yml` を作成します。

```yaml
outputDir: ./.codex/skills
targets:
  - url: https://github.com/rchaser53/skills-playground/tree/main/publish/summarize-website
    name: summarize-website
    version: v1.0.0
  - url: git@github.com:org/private-skill.git
    name: my-private-skill
    version: 4f3c2b1a9e6d7c8b0a1f2e3d4c5b6a7980fedcba
```

次を実行します。

```bash
go run .
```

既存ディレクトリを残したまま未インストールの skill だけ追加したい場合は `--skip-existing` を付けます。

```bash
go run . --skip-existing
```

YAML の `outputDir` より CLI 引数を優先したい場合は第 2 引数に渡します。

```bash
go run . ./codex-skills.yml ./.codex/skills
```

## 入力ファイル形式

入力は YAML です。次の形式に対応しています。

```yaml
outputDir: ./.codex/skills
targets:
  - url: https://github.com/rchaser53/skills-playground/tree/main/publish/summarize-website
    name: summarize-website
    version: v1.0.0
  - url: git@github.com:org/private-skill.git
    name: my-private-skill
    version: 4f3c2b1a9e6d7c8b0a1f2e3d4c5b6a7980fedcba
```

各要素の扱いは次のとおりです。

- `outputDir`: 省略可能。未指定時は `$PWD/.codex/skills`
- `targets[].url`: インストール元の Git リポジトリ URL。GitHub の tree URL を指定した場合はそのディレクトリをコピー
- `targets[].name`: インストール先ディレクトリ名
- `targets[].version`: 省略可能。branch、tag、または 40 文字の commit hash。未指定時は `main` を取得

`targets[].version` を指定した場合は、URL 側に含まれる branch/tag よりこちらを優先します。未指定時も `main` として扱います。GitHub の tree URL ではディレクトリ部分だけを再利用し、取得する ref は `version` に切り替わります。

## 対応している URL

- GitHub リポジトリ URL 例: `https://github.com/org/repo`
- GitHub ディレクトリ URL 例: `https://github.com/org/repo/tree/main/path/to/dir`
- Git リポジトリ URL 例: `https://github.com/org/repo.git`
- SSH 形式 例: `git@github.com:org/private-skill.git`

## 対応していない入力

- `SKILL.md` 単体への URL
- zip アーカイブ URL
- `targets` 配列以外の任意の YAML オブジェクト

## 動作仕様

1. URL 一覧ファイルを読み取る
2. 各リポジトリを一時ディレクトリに shallow clone する
3. `version` があれば、その branch/tag/commit hash を checkout する
4. GitHub の tree URL の場合は指定ディレクトリをそのままコピーする
5. それ以外はリポジトリ内の `SKILL.md` を探索する
6. skill ディレクトリまたは指定ディレクトリを `.codex/skills/<name>` にコピーする
7. 同名ディレクトリが既にあれば置き換える
8. `--skip-existing` 指定時は、同名ディレクトリが既にある skill をスキップする

## 使い方

ヘルプを表示する場合:

```bash
go run . --help
```

引数の仕様:

```text
go run . [options] [url_list_file] [install_dir]
```

- `--skip-existing`: インストール先に同名ディレクトリが既にある場合は削除せずスキップ
- `url_list_file`: 省略時は `./codex-skills.yml` を使用し、存在しない場合は `./codex-skils.yml` をフォールバックとして参照
- `install_dir`: 省略時は YAML の `outputDir` を使用し、さらに未指定なら `$PWD/.codex/skills`

## 注意点

- インストール先に同名ディレクトリがある場合は削除して再配置します
- `--skip-existing` を指定した場合は、同名ディレクトリがある skill をそのまま残してスキップします
- リポジトリ内に `SKILL.md` が見つからない場合はエラーになります
- private repository を使う場合は、その URL 形式に応じた認証設定が必要です

## リポジトリ構成

```text
.
├── README.md
├── go.mod
├── go.sum
└── main.go
```