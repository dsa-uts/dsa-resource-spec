# Resource 仕様

課題を作成するための `resource.yaml` のリファレンス。定義と参照素材を一つのディレクトリに置く。

[公開手順](publishing.md) · [Backend・Judge の実行規則](runtime.md) · [JSON Schema](../schemas/resource.schema.json)

## 最小例

素材を含む例は [testdata/resource/valid/basic/resource.yaml](../testdata/resource/valid/basic/resource.yaml) を参照。

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

- Resource・Workflow・Job の ID と成果物名は `^[a-z][a-z0-9-]*$`（英小文字で始まる英小文字・数字・ハイフン）。Workflow・Job の map key は ID、`name` は表示名。
- 未知フィールド、重複した YAML キー、複数の YAML 文書は拒否する。`schema-version` は持たない。
- `resource.version` は必須の SemVer 文字列（例: `"v1.0.0"`、省略形 `v1` / `v1.2` は不可）。公開時の更新規則は [公開手順](publishing.md#公開と再実行) を参照。
- 相対 path は clean POSIX 形式とし、空文字、`.`、`..` component、絶対 path、backslash、NUL、colon、空 component を禁止する。下表の root 外を指してはいけない。symlink と非 regular file を拒否し、OS が link count を提供する場合は hardlink も拒否する。

| パス | 基準となる場所 |
| --- | --- |
| `description-path`、`presets.files[].source`、`stdin.path`、`expected.*.path` | `resource.yaml` を置いた 課題ディレクトリ |
| `presets.files[].path` | 読み取り専用の `/preset` |
| `artifacts.*[].path` | 作業領域（`/workspace`） |

## 課題情報

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `resource.id` | 必須 | Resource の安定 ID。manifest を使う場合はその `id` と一致する。 |
| `resource.name` | 必須 | 課題の表示名。 |
| `resource.version` | 必須 | `vMAJOR.MINOR.PATCH` 形式の公開版。プレリリース・ビルドメタデータを許可する。 |
| `workflows` | 必須 | Workflow ID を key にした、1 個以上の Workflow の map。 |

## Workflow

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `name` | 任意 | 表示名。 |
| `description-path` | 任意 | 課題説明 Markdown などの path。 |
| `presets.files` | 任意 | Workflow 実行前に read-only `/preset` mount へ配置する Resource file。 |
| `jobs` | 必須 | Job ID をキーにした、1 個以上の Job の map。 |

### 配布素材（Preset）

```yaml
presets:
  files:
    - source: presets/Makefile
      path: Makefile
```

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `source` | 必須 | 課題ルート からの相対 path。 |
| `path` | 必須 | 読み取り専用の `/preset` 内の配置先 path。 |

同一 Workflow の `presets.files` 内で `path` が重複する場合は 検証エラー。

Preset は変更できない `/preset` に配置する。secret ではないため、非公開のテスト入力は `stdin.path` を使う。配置と保護の規則は [ファイルの配置](runtime.md#ファイルの配置と回収) を参照。

## Job

Job は独立した sandbox で実行する。同じ Job の Step は作業領域を共有するが、Job 間では共有しない。

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `name` | 任意 | 表示名。 |
| `visibility` | 任意 | `public` または `private`。省略時 `private`(既定で非公開)。 |
| `depends` | 任意 | 先行して完了している必要がある Job ID 配列。省略時 `[]`。 |
| `sandbox-image` | 必須 | タグまたは digest を含む完全なイメージ参照。 |
| `working-directory` | 任意 | Step の作業ディレクトリ。`/workspace` またはその配下の絶対パス。省略時 `/workspace`。 |
| `limits` | 必須 | 使用量と実行時間の上限。 |
| `artifacts` | 任意 | Job 間で明示的に受け渡す Artifact。 |
| `steps` | 必須 | 1 個以上の Step を実行順に並べた配列。 |

`working-directory` の各階層名は英数字・`_`・`.`・`-` のみ。`.`・`..` の階層や末尾の `/` は不可。

実行権限と結果の公開範囲は [実行規則](runtime.md#実行権限と順序) を参照。

### 実行イメージ

Job の `sandbox-image` は `ghcr.io/example/sandbox:latest` や `ghcr.io/example/sandbox@sha256:<digest>` の形式を使う。レジストリを含み、タグまたは digest が必須。validator はレジストリへ接続しない。公開時の digest 固定は [公開手順](publishing.md#公開と再実行) を参照。

## ジョブ間のファイル受け渡し

Job 間の受け渡しは宣言された 成果物ファイル のみ。次は Artifact 設定の抜粋（Job の必須項目は省略）。

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
| `artifacts.inputs[].from-job` | 必須 | 成果物を生成する Job の ID。同一 Workflow 内のみ指定可。 |
| `artifacts.inputs[].name` | 必須 | 生成元 Job の成果物名。 |
| `artifacts.inputs[].path` | 必須 | 作業領域（`/workspace`） 内の配置先の通常ファイルのパス。 |
| `artifacts.outputs[].name` | 必須 | 同一 Job 内で一意な Artifact name。 |
| `artifacts.outputs[].path` | 必須 | 作業領域（`/workspace`） 内の回収元の通常ファイルのパス。 |
| `artifacts.outputs[].visibility` | 任意 | `public` または `private`。省略時 `private`(既定で非公開)。 |
| `artifacts.outputs[].content-type` | public の場合 | 配信時の `Content-Type`。`private` では指定禁止。 |

### 公開する成果物

`visibility: public` の Artifact はクライアントに配信されうる。`content-type` は次の許可リストのみ:

- `image/png`
- `image/jpeg`
- `text/plain`
- `application/json`

SVG は script を実行できるため(stored XSS)public Artifact として許可しない。

### 依存関係

- `depends` は同一 Workflow 内の Job ID 配列。存在しない Job ID、自分自身、cycle は 検証エラー。
- `public` Job は `private` Job に `depends` してはいけない。`private` Job は `public` / `private` どちらの Job にも `depends` できる。
- `artifacts.inputs[].from-job` は現在の Job の `depends` に直接含まれていなければならない。`depends` していない Job、未実行 Job、自分自身の Artifact 参照は 検証エラー。
- `public` Job は `private` Job が生成した Artifact を入力にできない。`private` Job は `public` Job が生成した Artifact を入力にできる。

## 実行制限

`memory` と各 `*-size` は正の整数に `KiB`・`MiB`・`GiB` を付ける。小数や単位なしの指定は不可。

```yaml
limits:
  cpu: 1
  memory: 512MiB
  pids: 128
  step-timeout: "10s"
  stdout-size: 1MiB
  stderr-size: 1MiB
  workspace-size: 256MiB
  artifact-size: 1MiB
```

| フィールド | 必須 | 内容 |
| --- | --- | --- |
| `cpu` | 任意 | 当面 `1` 固定。指定する場合も `1` のみ許可。 |
| `memory` | 必須 | Job のメモリ上限。例: `512MiB`。 |
| `pids` | 任意 | 最大プロセス数。1 以上の整数。 |
| `step-timeout` | 必須 | Step timeout の既定値。例: `"2s"`、`"300ms"`。 |
| `stdout-size` | 任意 | stdout capture 上限。 |
| `stderr-size` | 任意 | stderr capture 上限。 |
| `workspace-size` | 任意 | 作業領域（`/workspace`） の容量上限。省略時 `256MiB`。 |
| `artifact-size` | 任意 | 1 成果物ファイル あたりの保存上限。省略時 `1MiB`。 |

### タイムアウトの書式

timeout は正の整数に `s`（秒）または `ms`（ミリ秒）を付けた文字列で指定する（例: `"2s"`、`"300ms"`）。数値のみ、0、負数、小数、空白付きの指定は許可しない。

Step の実効 timeout は `step.timeout`、省略時は `job.limits.step-timeout`。`limits.step-timeout` は全 Step に上書きがあっても必須。
Job 全体の時間制限は [実行時のタイムアウト](runtime.md#タイムアウト) を参照。

## Step

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
| `run` | 必須 | Bash script 文字列。複数行可。空・空白のみ・NUL byte・argv 配列は禁止。 |
| `compile` | 任意 | boolean。省略時 `false`。`true` の Step が失敗した場合は CE (Compilation Error) として記録する。 |
| `stdin` | 任意 | Step の標準入力。 |
| `timeout` | 任意 | Step の実時間による制限。例: `"2s"`、`"300ms"`。省略時は `limits.step-timeout`。 |
| `expected` | 任意 | 期待結果。 |

`compile: true` は失敗時のステータスを CE にする指定であり、`expected` 等による成功・失敗の判定条件は変更しない。`false` または省略時は通常の失敗判定を使う。ステータスの判定・記録は Judge が行う。

`expected.exit-code` は `0..255` の整数で、省略時 `0`。`expected.stdout` と `expected.stderr` は省略時、比較しない。

### 標準入力と期待する出力

`stdin`、`expected.stdout`、`expected.stderr` は、`value`（文字列を直接記述）か `path`（課題ルートからのファイルパス）のどちらか一つを必須とする。空文字列は `value: ""` と書く。

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

`stdin.path` の内容は Judge が標準入力に流し、期待値ファイルは Judge 内で比較に使う。どちらも 作業領域（`/workspace`） に配置しない。
