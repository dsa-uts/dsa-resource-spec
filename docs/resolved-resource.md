# 解決済み Resource

`LoadManifest` は YAML の入力用の型を公開せず、参照解決・単位変換・既定値補完済みの値を返す。`Manifest.Resources` は登録順の `[]Resource`、`SandboxImages` は CI 用のビルド設定。元の課題ディレクトリは保持しない。

`Resource` の JSON は `encoding/json.Marshal` で生成でき、`DecodeResource(io.Reader)` で復元・検証できる。CLI の `inspect <manifest-dir> <resource-id>` も同じ JSON を出力する。

| Go のフィールド | JSON キー・表現 |
| --- | --- |
| `Resource.Metadata` | `metadata`: `id`、`name`、`version` |
| `Resource.Workflows` | `workflows`: ID をキーにする map |
| `Workflow.Description` | `description`: Markdown の文字列 |
| `Workflow.Presets` | `presets`: `path`（配置先）、`content`（Base64）、`executable`（boolean）を持つ配列（未指定は `null`） |
| `Workflow.Jobs` | `jobs`: ID をキーにする map。順序は依存関係で決まる |
| `Job.Limits` | `limits`: `cpu`、`pids` は整数、`memory` と各 `*-size` はバイト数の整数（Go では `int64`） |
| `Step.Timeout` | `timeout`: ナノ秒の整数（Go では `time.Duration`）。2秒は `2000000000` |
| `Step.Stdin` | `stdin`: Base64（未指定は `null`） |
| `Step.Expected` | `expected`: `exit-code`、`stdout`、`stderr` |
| `OutputExpectation` | `content`: Base64、`match`: `exact` / `easy` / `sorted` |

YAML と同じ意味を持つその他のフィールドは同じ kebab-case のキーを使う。`Step.Compile` は省略時の false、Job の visibility は public、成果物の visibility は private を明示的に保持する。期待終了コードは省略時 0。実効 timeout は各 Step に保持し、`Limits.StepTimeout` は持たない。

期待出力の `null` は比較しない。`{"content":"","match":"exact"}` は空の出力を期待する。`[]byte` の nil と空はともに空の内容として利用できるが、期待出力オブジェクト自体の有無には意味がある。

サイズの既定値は workspace 128 MiB、artifact 1 MiB、stdout / stderr 各 10 MiB。CPU は 1、PIDs は 128。これらは明示指定の上限ではない（CPU は現在 1 固定）。メモリと timeout は YAML で指定必須。

`DecodeResource` はファイルを読まず、単位や既定値を再解釈しない。正の実行制限、ID・バージョン、イメージ参照、配置先、依存関係・循環、成果物参照、期待結果を検証する。未知フィールドはネスト内も拒否し、旧 `definition`、`files`、参照元の `source` や `description-path` を受け付けない。

ビルド設定は Manifest にだけ含む。`context` / `dockerfile` / `image` / `platforms` を保持し、Dockerfile やビルドコンテキストの bytes は含まない。
