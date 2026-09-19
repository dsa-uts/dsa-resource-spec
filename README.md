# dsa-resource-spec

課題定義 `resource.yaml` を検証・読み込み・ZIP 化する Go ライブラリと CLI。

| やりたいこと | 読む文書 |
| --- | --- |
| 課題の定義を書く | [Resource 仕様](docs/resource.md) |
| 課題を登録して公開する | [公開手順](docs/publishing.md) |
| Backend・Judge を実装する | [実行規則](docs/runtime.md) |
| 設計の理由を知る | [設計](docs/design.md) |

## 開発環境

Nix と direnv をインストールし、direnv の shell hook を設定した環境で実行します。

```sh
direnv allow
check
```

`check` は Go の vet・テストと Python のテストを実行します。direnv を使わない場合は `nix develop path:.` で同じ devshell に入れます。

Go は 1.27 系 stable（lock 時点で 1.27.1）。`flake.lock` で環境を固定し、`GOTOOLCHAIN=local` で Nix の Go を使用します。macOS/Linux の arm64・amd64 に対応します。

## CLI で試す

リポジトリルートで、同梱の課題を検証できます。

```sh
go run ./cmd/resource-spec validate testdata/valid
go run ./cmd/resource-spec inspect testdata/valid
```

`validate` は検証、`inspect` は検証済み定義の JSON 出力です。公開に使う `manifest`・`archive` は [公開手順](docs/publishing.md) を参照してください。`compare VERSION VERSION` は公開処理用に SemVer の大小を `-1`・`0`・`1` で返します。

## Go から使う

```go
import (
    "os"
    resource "github.com/dsa-uts/dsa-resource-spec"
)

func load() (*resource.Resource, error) {
    return resource.Read(os.DirFS("testdata/valid"))
}
```

戻り値の `Definition` は型付き定義、`Files` は課題ルートからの相対パスをキーとする素材の bytes です。`archive/zip.Reader` も `fs.FS` として渡せます。

| API | 用途 |
| --- | --- |
| `Read(fs.FS) (*Resource, error)` | 定義を検証し、説明文・Preset・標準入力・期待出力の参照ファイルを一括で読む。入力 filesystem は保持しない。 |
| `Validate(fs.FS) error` | `Read` と同じ検証だけを行う。 |
| `Archive(fs.FS, io.Writer, map[string]string) error` | 解決済みイメージ参照を使って課題全体を ZIP 化する。 |
| `ReadManifest(fs.FS) (*Manifest, error)` | 作者・CI 用の課題一覧とビルド入力を検証する。配布 ZIP の読み込みには不要。 |

取得・認証・展開・採点・実行・公開済みバージョン管理は呼び出し側で行います。

### 入力 filesystem の条件

入力は呼び出し中に変更しないでください。symlink、非 regular file、OS が link count を提供する場合の hardlink を拒否します。独自の `fs.FS` はファイル種別を正しく公開する必要があります。同時書き換えや、隠蔽された hardlink の検出は保証しません。ZIP の取得・展開サイズの上限は取り込み側で設定してください。
