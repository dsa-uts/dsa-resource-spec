# dsa-resource-spec

課題定義 `resource.yaml` を検証・読み込みする Go ライブラリと CLI。実際の課題は `ex1/` に置き、mainのCIでsandboxイメージと `release/` の課題JSONを公開する。

| やりたいこと | 読む文書 |
| --- | --- |
| 課題の定義を書く | [Resource 仕様](docs/resource.md) |
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

| 関数 | 用途 |
| --- | --- |
| `LoadManifest(dir string) (*Manifest, error)` | `manifest.yaml` と全課題を検証し、素材の読み込み・単位変換・既定値の補完を行う。 |
| `Resource.Hash() (string, error)` | 呼び出し時点の Resource の JSON から SHA-256 を計算する。 |
| `DecodeResource(r io.Reader) (*Resource, error)` | 解決済み Resource の JSON を復元し、依存関係・実行制限などを検証する。未知フィールドと複数文書は拒否する。 |
