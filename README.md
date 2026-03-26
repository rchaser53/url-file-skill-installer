# codex-skills-installer

YAML ファイルに並べた Git リポジトリ URL から、Codex skill を `.codex/skills` 配下へまとめてインストールするための小さなユーティリティです。

このリポジトリには次の 2 つが含まれます。

- `SKILL.md`: このインストーラ自体を Codex skill として説明する定義ファイル
- `scripts/install_skills_from_url_file.sh`: URL 一覧ファイルを読み取り、skill をインストールするシェルスクリプト

## できること

- YAML の配列から skill リポジトリ URL を読み取る
- `URL -> skill_name` 形式でインストール先ディレクトリ名を指定できる
- GitHub URL と一般的な Git リポジトリ URL に対応する
- 対象リポジトリ内の `SKILL.md` を検出して `.codex/skills` に配置する

## 前提条件

- macOS / Linux などの Bash が使える環境
- `git` コマンドが利用可能であること

## クイックスタート

プロジェクトルートに `codex-skills.yml` を作成します。

```yaml
skills:
  - "https://github.com/rchaser53/summarize-website"
  - "git@github.com:org/private-skill.git -> my-private-skill"
```

次を実行します。

```bash
bash scripts/install_skills_from_url_file.sh
```

インストール先を明示したい場合は第 2 引数に渡します。

```bash
bash scripts/install_skills_from_url_file.sh ./codex-skills.yml ./.codex/skills
```

## 入力ファイル形式

入力は YAML です。以下の 2 形式に対応しています。

### 1. トップレベル配列

```yaml
- "https://github.com/rchaser53/summarize-website"
- "git@github.com:org/private-skill.git -> my-private-skill"
```

### 2. `skills` キー配下の配列

```yaml
skills:
  - "https://github.com/rchaser53/summarize-website"
  - "git@github.com:org/private-skill.git -> my-private-skill"
```

各要素の扱いは次のとおりです。

- `URL`: リポジトリ名を元にインストール
- `URL -> skill_name`: 指定した名前でインストール

空行と `#` で始まるコメント行は無視されます。

## 対応している URL

- GitHub リポジトリ URL 例: `https://github.com/org/repo`
- Git リポジトリ URL 例: `https://github.com/org/repo.git`
- SSH 形式 例: `git@github.com:org/private-skill.git`

## 対応していない入力

- `SKILL.md` 単体への URL
- zip アーカイブ URL
- 文字列配列ではない任意の YAML オブジェクト

## 動作仕様

1. URL 一覧ファイルを読み取る
2. 各リポジトリを一時ディレクトリに shallow clone する
3. リポジトリ内の `SKILL.md` を探索する
4. skill ディレクトリを `.codex/skills/<name>` にコピーする
5. 同名ディレクトリが既にあれば置き換える

## 使い方

ヘルプを表示する場合:

```bash
bash scripts/install_skills_from_url_file.sh --help
```

引数の仕様:

```text
install_skills_from_url_file.sh [url_list_file] [install_dir]
```

- `url_list_file`: 省略時は `./codex-skills.yml` を使用し、存在しない場合は `./codex-skils.yml` をフォールバックとして参照
- `install_dir`: 省略時は `$PWD/.codex/skills`

## 注意点

- インストール先に同名ディレクトリがある場合は削除して再配置します
- リポジトリ内に `SKILL.md` が見つからない場合はエラーになります
- private repository を使う場合は、その URL 形式に応じた認証設定が必要です

## リポジトリ構成

```text
.
├── README.md
├── SKILL.md
└── scripts/
    └── install_skills_from_url_file.sh
```