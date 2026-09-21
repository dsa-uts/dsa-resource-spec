# dsa-resource-spec

課題定義 `resource.yaml` の検証・読み込みと出力比較を提供する Go ライブラリと CLI。

| やりたいこと | 読む文書 |
| --- | --- |
| 課題の定義を書く | [リソース仕様](docs/resource.md) |
| 課題を登録してイメージをビルドする | [登録・ビルド手順](docs/publishing.md) |
| Backend・Judge を実装する | [実行規則](docs/runtime.md) |
| 設計の理由を知る | [設計](docs/design.md) |

## 開発環境

* Go 1.27

Goの環境はNix, direnvで設定することも可。

```
direnv allow .
check
```

## CLI

```sh
# manifest.yaml と登録された全課題を検証する
go run ./cmd/resource-spec validate <manifest-dir>
# マニフェストと登録された全課題のメタデータを JSON 出力する
go run ./cmd/resource-spec catalog <manifest-dir>
# 指定した課題の解決済み JSON を出力する
go run ./cmd/resource-spec show <manifest-dir> <resource-id>
```

GitHub Actions 用の検証・公開処理は、別CLIの `resource-ci` に分離している。使い方は [公開手順](docs/publishing.md) を参照。

### CLI のリリース

[CLI Release workflow](.github/workflows/cli-release.yml) をデフォルトブランチに追加後、
Actions の **CLI Release → Run workflow** で、push 済みのタグ（例: `v1.1.0`）を指定する。
指定タグのソースを検証・ビルドして GitHub Release を作成する。既存タグの付け替えは不要。
タグは `vMAJOR.MINOR.PATCH` 形式に対応し、既存 Release の上書きは行わない。

Linux / macOS の amd64 / arm64 向けに、`resource-spec`・`resource-ci`・LICENSE をまとめた
`resource-cli-<tag>-<os>-<arch>.tar.gz` と `checksums.txt` を配布する。
利用側はバージョンを固定してダウンロードし、チェックサムを照合して展開した CLI を使う。
CLI の実行に Go は不要だが、課題公開には引き続き Git・Docker Buildx・regctl が必要。
課題 JSON の `release/` への公開とは別の処理である。

## Go から使う

```go
import resource "github.com/dsa-uts/dsa-resource-spec"
```

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

| 関数 | 用途 |
| --- | --- |
| `LoadManifest(dir string) (*Manifest, error)` | `manifest.yaml` と全課題を検証し、参照ファイルの読み込み・単位変換・既定値の補完を行う。 |
| `Resource.Hash() (string, error)` | 呼び出し時点の Resource の JSON から SHA-256 を計算する。 |
| `DecodeResource(r io.Reader) (*Resource, error)` | 解決済み Resource の JSON を復元し、依存関係・実行制限などを検証する。未知フィールドと複数文書は拒否する。 |
| `MatchOutput(actual, expected []byte, mode MatchMode) (bool, error)` | `MatchExact`・`MatchEasy`・`MatchSorted` で出力を比較する。未知・空のモードはエラー。不一致と不正な UTF-8 は `false, nil`。 |

```go
if want := step.Expected.Stdout; want != nil {
    matched, err := resource.MatchOutput(stdout, want.Content, want.Match)
    // err は設定エラー、matched は比較結果として扱う。
}
```

期待値が省略されている場合は呼び出し側で比較を省く。
詳しい比較規則は [出力の比較](docs/runtime.md#出力の比較) を参照。
