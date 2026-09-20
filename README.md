# dsa-resource-spec

課題定義 `resource.yaml` を検証・読み込みする Go ライブラリと CLI。

| やりたいこと | 読む文書 |
| --- | --- |
| 課題の定義を書く | [Resource 仕様](docs/resource.md) |
| 課題を登録してイメージをビルドする | [登録・ビルド手順](docs/publishing.md) |
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
# manifest.yaml と登録された全課題を検証する
resource-spec validate <manifest-dir>
# 素材を解決したマニフェスト全体を JSON 出力する
resource-spec manifest <manifest-dir>
# 指定した課題の解決済み JSON を出力する
resource-spec inspect <manifest-dir> <resource-id>
```

チェックアウトからは `go run ./cmd/resource-spec` でも実行できます。

```sh
go run ./cmd/resource-spec inspect testdata/resource/valid/basic sample
```

## Go から使う

```go
manifest, err := resource.LoadManifest("path/to/manifest-directory")
if err != nil {
    return err
}
// Resources は manifest.yaml の登録順。ID は Metadata.ID にある。
data, err := json.Marshal(manifest.Resources[0])
if err != nil {
    return err
}
restored, err := resource.DecodeResource(bytes.NewReader(data))
```

パッケージの import path は `github.com/dsa-uts/dsa-resource-spec`（package 名 `resource`）です。

| 関数 | 用途 |
| --- | --- |
| `LoadManifest(dir string) (*Manifest, error)` | `manifest.yaml` と全課題を検証し、素材の読み込み・単位変換・既定値の補完を行う。 |
| `DecodeResource(r io.Reader) (*Resource, error)` | 解決済み Resource の JSON を復元し、依存関係・実行制限などを検証する。未知フィールドと複数文書は拒否する。 |

公開データは [解決済み JSON の契約](docs/resolved-resource.md) を参照してください。説明文は Markdown の `string`、その他の素材は `[]byte` です。読み込み後にファイルへアクセスする必要はありません。

### ローカルファイルの読み込み

入力は呼び出し中に変更しないでください。素材の参照は各 `resource.yaml` のディレクトリを基準とし、`../shared/input.txt` のような共有素材を許可します。`os.Root` によりアクセスをマニフェストのディレクトリ内に制限します。範囲内を指す相対 symlink は辿り、範囲外と絶対 symlink は拒否します。

定義と明示的に参照した通常ファイルだけを読み込みます。隠しファイルや `node_modules` の一律除外は行いません。未参照ファイル、Dockerfile、ビルドコンテキストは読みません。Preset はリンク先の内容と実行ビットを取り込み、元の参照パスは返しません。hardlink は通常ファイルとして扱います。

同時書き換えの検出や一時点のスナップショット取得は保証しません。入力容量、取得・認証、採点・実行、公開済みバージョン管理は呼び出し側で管理してください。
