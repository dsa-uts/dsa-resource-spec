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

戻り値の `Definition` は型付き定義、`Files` は課題ルートからの相対パスをキーとする素材の bytes です。`archive/zip.Reader` も `fs.FS` として渡せます。入力形式の判定や ZIP を開く・閉じる処理は呼び出し側で行い、ディスクへの展開は不要です。

| API | 用途 |
| --- | --- |
| `Read(fs.FS) (*Resource, error)` | 課題内の通常ファイルを一括で取り込み、定義と参照素材を検証して返す。入力 filesystem は保持しない。 |
| `Validate(fs.FS) error` | `Read` と同じ検証だけを行う。 |
| `ReadManifest(fs.FS) (*Manifest, error)` | 作者・CI 用の課題一覧とビルド設定(resources.yaml)を検証する|

取得・認証・展開・採点・実行・公開済みバージョン管理は呼び出し側で行います。

### 入力 filesystem の条件

入力は呼び出し中に変更しないでください。`Read` は課題内を列挙し、symlink を辿らずに除外して、残った通常ファイルを未参照のものも含めてメモリに取り込みます。定義と参照素材の検証は、その後メモリ上だけで行います。除外されたファイルが定義・参照素材として必要なら、ファイル不存在のエラーになります。戻り値の `Files` は参照素材のみです。

ディレクトリ以外の非 regular file と、OS が link count を提供する場合の hardlink は拒否します。独自の `fs.FS` はファイル種別を正しく公開する必要があります。同時書き換えや、隠蔽された hardlink の検出は保証しません。未参照ファイルも取り込むため、課題ディレクトリには配布に必要なファイルを置き、入力の総容量や ZIP の取得・展開サイズの上限は呼び出し側で管理してください。

`ReadManifest` は `resources.yaml` と登録された各課題だけを読みます。manifest 自体は信頼できる CI 入力として通常のファイル読み込みを行います。各課題ディレクトリまでのパスにある symlink は拒否し、課題内の検証は `Read` に任せます。
