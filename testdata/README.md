# 検証 fixture

各ケースは `manifest.yaml` と登録された課題を含む、単独で読み込めるディレクトリです。

- `resource/valid/<case>/`: 正常な課題。`sample/` 内に定義と素材を配置します。
- `resource/invalid/<case>/`: 不正な課題。意図した不備（素材の欠落など）をそのまま配置します。
- `manifest/valid/<case>/`: 正常なマニフェスト。
- `manifest/invalid/<case>/`: ID 不一致や YAML キー重複などの不正なマニフェスト。

```sh
go run ./cmd/resource-spec validate testdata/resource/valid/basic
go run ./cmd/resource-spec inspect testdata/resource/valid/basic sample
go run ./cmd/resource-spec validate testdata/resource/invalid/missing-expected
go run ./cmd/resource-spec manifest testdata/manifest/valid/basic
```

`valid` は成功、`invalid` は失敗を期待します。ライブラリと CLI のテストは各ケースを読み込みます。

`symlink`、`directory-symlink`、`definition-symlink` は範囲内を指す相対 symlink を含む正常系です。checkout 時にも symlink を保持してください。範囲外への参照、共有素材、実行ビット、JSON の往復などは一時ディレクトリを使うテストで検証します。
