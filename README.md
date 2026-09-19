# dsa-resource-spec

課題定義 `resource.yaml` を検証・読み込みする Go ライブラリと CLI。

| やりたいこと | 読む文書 |
| --- | --- |
| 課題の定義を書く | [Resource 仕様](docs/resource.md) |
| 課題を登録して公開する | [公開手順](docs/publishing.md) |
| Backend・Judge を実装する | [実行規則](docs/runtime.md) |
| 設計の理由を知る | [設計](docs/design.md) |

## 開発環境

* Go 1.27
* Python 3.14

GoやPythonの環境はNix, direnvで設定することも可。

```
direnv allow .
check
```

## CLI 

Linux 用バイナリ（amd64・arm64）の取得・公開は [CLI のリリース](docs/cli-release.md) を参照してください。

```sh
# validate <resource dir>: リソース定義を検証する
go run ./cmd/resource-spec validate testdata/resource/valid/basic
# inspect <resource dir>: リソース定義を検証して、読み込んだ結果をJSONで出力する
go run ./cmd/resource-spec inspect testdata/resource/valid/basic
```

公開に使う `manifest` は [公開手順](docs/publishing.md) を参照してください。`compare VERSION VERSION` は公開処理用に SemVer の大小を `-1`・`0`・`1` で返します。

## Go から使う

```go
import (
    "os"
    resource "github.com/dsa-uts/dsa-resource-spec"
)

func load() (*resource.Resource, error) {
    return resource.Read(os.DirFS("testdata/resource/valid/basic"))
}
```

戻り値の `Definition` は型付き定義、`Files` は課題ルートからの相対パスをキーとする素材の bytes です。`archive/zip.Reader` も `fs.FS` として渡せます。

| API | 用途 |
| --- | --- |
| `Read(fs.FS) (*Resource, error)` | 定義を検証し、説明文・Preset・標準入力・期待出力の参照ファイルを一括で読む。入力 filesystem は保持しない。 |
| `Validate(fs.FS) error` | `Read` と同じ検証だけを行う。 |
| `ReadManifest(fs.FS) (*Manifest, error)` | 作者・CI 用の課題一覧とビルド設定を検証する。配布 ZIP の読み込みには不要。 |

取得・認証・展開・採点・実行・公開済みバージョン管理は呼び出し側で行います。

### 入力 filesystem の条件

入力は呼び出し中に変更しないでください。symlink、非 regular file、OS が link count を提供する場合の hardlink を拒否します。独自の `fs.FS` はファイル種別を正しく公開する必要があります。同時書き換えや、隠蔽された hardlink の検出は保証しません。ZIP の取得・展開サイズの上限は取り込み側で設定してください。
