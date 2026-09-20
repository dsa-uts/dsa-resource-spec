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

戻り値の `Definition` は型付き定義、`Files` は課題ルートからの相対パスをキーとする素材の bytes です。`archive/zip.Reader` も `fs.FS` として渡せます。入力形式の判定や ZIP を開く・閉じる処理は呼び出し側で行い、ディスクへの展開は不要です。

| API | 用途 |
| --- | --- |
| `Read(fs.FS) (*Resource, error)` | 課題内の通常ファイルを一括で取り込み、定義と参照素材を検証して返す。入力 filesystem は保持しない。 |
| `Validate(fs.FS) error` | `Read` と同じ検証だけを行う。 |
| `ReadManifest(fs.FS) (*Manifest, error)` | 作者・CI 用の課題一覧とビルド設定(resources.yaml)を検証する|

取得・認証・展開・採点・実行・公開済みバージョン管理は呼び出し側で行います。

### 入力 filesystem の条件

入力は呼び出し中に変更しないでください。`Read` と `ReadManifest` は入力全体を一度だけ走査し、通常ファイルを未参照のものも含めてメモリに取り込みます。全階層で symlink、`.` で始まるファイル・ディレクトリ、`node_modules` ディレクトリを除外します。symlink は辿らず、除外ディレクトリ内は走査しません。定義・参照素材が除外された場合は、ファイル不存在のエラーになります。

取り込み後はメモリ上だけで検証します。`ReadManifest` は同じデータを共有して登録された全課題を検証し、課題ごとの再走査やファイル内容の再コピーは行いません。`Read` の戻り値の `Files` は参照素材のみです。

hardlink は通常ファイルとして扱い、inode の共有は検査しません。除外対象以外の、ディレクトリではない非 regular file は拒否します。独自の `fs.FS` はファイル種別を正しく公開する必要があります。同時書き換えの検出や、ある一時点の内容の取得は保証しません。未参照ファイルも取り込むため、入力の総容量や ZIP の取得・展開サイズの上限は呼び出し側で管理してください。
