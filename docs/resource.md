# リソース仕様

課題を作成するための `resource.yaml` のリファレンス。定義を課題ディレクトリに置き、`manifest.yaml` に登録する。参照ファイルはマニフェストのディレクトリ内で共有できる。

[登録・ビルド手順](publishing.md) · [Backend・Judge の実行規則](runtime.md) · [JSON Schema](../schemas/resource.schema.json)

## 最小例

ファイルを参照する例は [testdata/cli/valid/basic/input/sample/resource.yaml](../testdata/cli/valid/basic/input/sample/resource.yaml) を参照。

```yaml
resource:
  id: example
  name: Example
  version: "v1.0.0"
workflows:
  judge:
    jobs:
      test:
        visibility: public
        sandbox-image: ghcr.io/example/sandbox:latest
        limits:
          memory: 512MiB
          step-timeout: "10s"
        steps:
          - run: 'printf "AC\n"'
            expected:
              stdout:
                match: exact
                value: "AC\n"
```

## 共通の規則

- リソース・ワークフロー・ジョブの ID と成果物名は `^[a-z][a-z0-9-]*$`（英小文字で始まる英小文字・数字・ハイフン）。ワークフロー・ジョブのマップのキーは ID、`name` は表示名。
- 未知フィールド、重複した YAML キー、複数の YAML 文書は拒否する。`schema-version` は持たない。
- `resource.version` は必須の SemVer 文字列（例: `"v1.0.0"`、省略形 `v1` / `v1.2` は不可）。
- ファイルの参照は `resource.yaml` のあるディレクトリ基準。`..` と範囲内を指す相対パスのシンボリックリンクを許可するが、マニフェストのディレクトリ外と絶対パスのシンボリックリンクは拒否する。絶対パス、バックスラッシュ、NUL、コロンは指定できない。明示的に参照された隠しファイルや `node_modules` 内のファイルも読み込む。
- Preset の配置先・成果物のパスは正規化済みの POSIX 形式の相対パスとし、空文字、`.`、`..` というパス要素、絶対パス、バックスラッシュ、NUL、コロン、空のパス要素を禁止する。
- 参照先は通常ファイルのみ。ハードリンクは通常ファイルとして扱い、inode の共有は検査しない。未参照ファイルは読み込まない。

| パス | 基準となる場所 |
| --- | --- |
| `description-path`、`presets.files[].source`、`stdin.path`、`expected.*.path` | `resource.yaml` を置いた課題ディレクトリ |
| `presets.files[].path` | 読み取り専用の `/preset` |
| `artifacts.*[].path` | 作業領域 |

## 課題情報

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `resource.id` | 必須 | リソースを識別する、バージョン間で共通の ID。マニフェストを使う場合はその `id` と一致する。 |
| `resource.name` | 必須 | 課題の表示名。 |
| `resource.version` | 必須 | `vMAJOR.MINOR.PATCH` 形式の公開バージョン。プレリリース・ビルドメタデータを許可する。 |
| `required-files` | 任意 | 課題全体の提出ファイルを案内する文字列配列。省略・`[]` は案内なし。 |
| `workflows` | 必須 | ワークフロー ID をキーとするマップ。1 件以上必須。 |

`required-files` は `resource`・`workflows` と同じトップレベルに置く。

```yaml
required-files:
  - main.c
  - "*.h"
  - レポート.pdf（任意）
```

各要素は空文字・空白のみを禁止し、それ以外は記載順・内容をそのまま保持する。
表示用の案内であり、パス検証・ワイルドカードの解釈・ファイルの読み込みは行わない。
提出可否やワークフローの実行判定には使わない。
公開 JSON ではトップレベルの `required-files` を必ず出力し、YAML で省略した場合も `[]` とする。

## ワークフロー

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `name` | 任意 | 表示名。 |
| `description-path` | 任意 | 課題説明 Markdown などのパス。 |
| `presets.files` | 任意 | ワークフロー実行前に読み取り専用の `/preset` に配置する配布ファイル。 |
| `jobs` | 必須 | ジョブ ID をキーとするマップ。1 件以上必須。 |

### 配布ファイル（Preset）

```yaml
presets:
  files:
    - source: presets/Makefile
      path: Makefile
```

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `source` | 必須 | 課題ディレクトリからの相対パス。 |
| `path` | 必須 | 読み取り専用の `/preset` 内の配置先パス。 |

読み込み時に `source` の内容と実行権限の有無（いずれかの実行ビットが設定されているか）を取り込む。

同一ワークフローの `presets.files` 内で `path` が重複する場合は検証エラー。

Preset は変更できない `/preset` に配置する。実行するプログラムから読み取れるため、非公開のテスト入力は `stdin.path` を使う。配置と保護の規則は [ファイルの配置](runtime.md#ファイルの配置と回収) を参照。

## ジョブ

ジョブは独立したサンドボックスで実行する。同じジョブのステップは作業領域を共有するが、ジョブ間では共有しない。

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `name` | 任意 | 表示名。 |
| `visibility` | 任意 | `public` または `private`。省略時 `public`（既定で公開）。 |
| `depends` | 任意 | 先行して完了している必要があるジョブ ID 配列。省略時 `[]`。 |
| `sandbox-image` | 必須 | タグまたはダイジェストを含む完全なイメージ参照。 |
| `limits` | 必須 | 使用量と実行時間の上限。 |
| `artifacts` | 任意 | ジョブ間で明示的に受け渡す成果物。 |
| `steps` | 必須 | 1 個以上のステップを実行順に並べた配列。 |

各ステップはジョブの作業領域をカレントディレクトリとして開始する。作業領域の絶対パスは Judge が決める。必要なら `run` 内で `cd` する。ステップ内の `cd` は次のステップに引き継がない。

実行権限と結果の公開範囲は [実行規則](runtime.md#実行権限と順序) を参照。

### 実行イメージ

ジョブの `sandbox-image` は `ghcr.io/example/sandbox:latest` や `ghcr.io/example/sandbox@sha256:<digest>` の形式を使う。レジストリを含み、タグまたはダイジェストが必須。検証処理はレジストリへ接続しない。

## ジョブ間のファイル受け渡し

ジョブ間の受け渡しは宣言された成果物ファイルのみ。次は成果物設定の抜粋（ジョブの必須項目は省略）。

```yaml
jobs:
  build:
    visibility: public
    artifacts:
      outputs:
        - name: program
          path: build/program
        - name: diagram
          path: out/diagram.png
          visibility: public
          content-type: image/png

  hidden-test:
    depends: [build]
    artifacts:
      inputs:
        - from-job: build
          name: program
          path: build/program
```

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `artifacts.inputs[].from-job` | 必須 | 成果物を生成するジョブの ID。同一ワークフロー内のみ指定可。 |
| `artifacts.inputs[].name` | 必須 | 生成元ジョブの成果物名。 |
| `artifacts.inputs[].path` | 必須 | 作業領域内の配置先の通常ファイルのパス。 |
| `artifacts.outputs[].name` | 必須 | 同一ジョブ内で一意な成果物名。 |
| `artifacts.outputs[].path` | 必須 | 作業領域内の回収元の通常ファイルのパス。 |
| `artifacts.outputs[].visibility` | 任意 | `public` または `private`。省略時 `private`（既定で非公開）。 |
| `artifacts.outputs[].content-type` | `public` の場合 | 配信時の `Content-Type`。`private` では指定禁止。 |

### 公開する成果物

`visibility: public` の成果物はクライアントに配信されることがある。`content-type` は次の値のみ指定できる。

- `image/png`
- `image/jpeg`
- `text/plain`
- `application/json`

SVG はスクリプトを実行でき、格納型 XSS の原因となるため、公開する成果物として許可しない。

### 依存関係

- `depends` は同一ワークフロー内のジョブ ID 配列。存在しないジョブ ID、自分自身、循環する依存関係は検証エラー。
- `public` ジョブは `private` ジョブに依存してはいけない。`private` ジョブは `public` / `private` どちらのジョブにも依存できる。
- `artifacts.inputs[].from-job` は現在のジョブの `depends` に直接含まれていなければならない。`depends` に含まれないジョブ、未実行ジョブ、自分自身の成果物参照は検証エラー。
- `public` ジョブは `private` ジョブが生成した成果物を入力にできない。`private` ジョブは `public` ジョブが生成した成果物を入力にできる。

## 実行制限

`memory` と各 `*-size` は正の整数に `KiB`・`MiB`・`GiB` を付ける。小数や単位なしの指定は不可。

```yaml
limits:
  cpu: 1
  memory: 512MiB
  pids: 128
  step-timeout: "10s"
  stdout-size: 4KiB
  stderr-size: 4KiB
  workspace-size: 256MiB
  artifact-size: 1MiB
```

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `cpu` | 任意 | CPU 上限。1 以上の整数。省略時 `1`。 |
| `memory` | 任意 | ジョブのメモリ上限。例: `512MiB`。省略時 `128MiB`。 |
| `pids` | 任意 | 最大プロセス数。1 以上の整数。省略時 `128`。 |
| `step-timeout` | 必須 | ステップのタイムアウトの既定値。例: `"2s"`、`"300ms"`。 |
| `stdout-size` | 任意 | 標準出力の取得サイズの上限。最大 `128KiB`。省略時 `4KiB`。 |
| `stderr-size` | 任意 | 標準エラー出力の取得サイズの上限。最大 `128KiB`。省略時 `4KiB`。 |
| `workspace-size` | 任意 | 作業領域の容量上限。省略時 `128MiB`。 |
| `artifact-size` | 任意 | 成果物ファイル 1 個あたりの保存上限。省略時 `1MiB`。 |

これらは省略時の既定値であり、指定可能な絶対上限ではない。標準出力・標準エラー出力に `20MiB` なども指定できる。サイズは符号付き 64 ビット整数のバイト数に収まる必要がある。

### タイムアウトの書式

タイムアウトは正の整数に `s`（秒）または `ms`（ミリ秒）を付けた文字列で指定する（例: `"2s"`、`"300ms"`）。数値のみ、0、負数、小数、空白付きの指定は許可しない。

ステップに実際に適用されるタイムアウト値は `step.timeout`、省略時は `job.limits.step-timeout`。`limits.step-timeout` はすべてのステップで個別に指定していても必須。
ジョブ全体の時間制限は [実行時のタイムアウト](runtime.md#タイムアウト) を参照。

## ステップ

```yaml
steps:
  - name: Test
    run: "make test"
    timeout: "60s"
    expected:
      exit-code: 0
      stdout:
        match: exact
        path: expected/test.stdout
      stderr:
        match: exact
        value: ""
```

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `name` | 任意 | 表示名。 |
| `run` | 必須 | Bash スクリプトの文字列。複数行可。空・空白のみ・NUL バイト・引数の配列は禁止。 |
| `compile` | 任意 | 真偽値。省略時 `false`。`true` のステップが失敗した場合は CE (Compilation Error) として記録する。 |
| `stdin` | 任意 | ステップの標準入力。 |
| `timeout` | 任意 | ステップの経過時間の上限。例: `"2s"`、`"300ms"`。省略時は `limits.step-timeout`。 |
| `expected` | 任意 | 期待結果。 |

`compile: true` は失敗時のステータスを CE にする指定であり、`expected` 等による成功・失敗の判定条件は変更しない。`false` または省略時は通常の失敗判定を使う。ステータスの判定・記録は Judge が行う。

`expected.exit-code` は `0..255` の整数で、省略時は終了コードをチェックしない。正常終了を期待する場合は `exit-code: 0` を明示する。`expected.stdout` と `expected.stderr` は省略時、比較しない。

### 標準入力と期待する出力

`stdin`、`expected.stdout`、`expected.stderr` は、`value`（文字列を直接記述）か `path`（課題ディレクトリからのファイルパス）のどちらか一つを必須とする。空文字列は `value: ""` と書く。

```yaml
stdin:
  path: input/sample.txt
expected:
  exit-code: 0
  stdout:
    match: exact
    value: "AC\n"
  stderr:
    match: easy
    path: expected/stderr.txt
```

期待出力には `match` も必須。`exact` は完全一致、`easy` は空白を正規化、`sorted` はさらに行内の要素順を無視する。厳密な手順は [比較規則](runtime.md#出力の比較) を参照。

`stdin.path` の内容は Judge が標準入力に流し、期待値ファイルは Judge 内で比較に使う。どちらも作業領域に配置しない。
