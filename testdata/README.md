# 検証 fixture

各ケースのディレクトリは、単独で検証できる完全なファイルシステムです。他のケースのファイルのコピーや、定義の差し替えは不要です。

- `resource/valid/<case>/`: `resource.yaml` と参照素材を含む正常な課題。
- `resource/invalid/<case>/`: 課題の異常系。意図した不備（素材の欠落など）もそのまま配置します。
- `manifest/valid/<case>/`: `resources.yaml` と参照先の課題を含む正常な manifest。
- `manifest/invalid/<case>/`: manifest の異常系。`id-mismatch` と `duplicate-yaml-key` を含みます。

```sh
go run ./cmd/resource-spec validate testdata/resource/valid/basic
go run ./cmd/resource-spec inspect testdata/resource/valid/basic
go run ./cmd/resource-spec validate testdata/resource/invalid/missing-expected
go run ./cmd/resource-spec manifest testdata/manifest/valid/basic
go run ./cmd/resource-spec manifest testdata/manifest/invalid/id-mismatch
```

`valid` は成功、`invalid` は失敗を期待します。ライブラリのテストは `os.DirFS` で各ケースを直接読み、CLI の E2E テストも同じディレクトリを渡して終了コードを検証します。ケースを追加すると両方のテストに自動で含まれます。

`symlink`、`directory-symlink`、`definition-symlink` は実際の相対 symlink を含み、すべてのリンク先は各ケース内にあります。リンクの除外によって必要なファイルが不存在となることを検証するため、checkout 時にも symlink を保持してください。Git は hardlink を保持しないため、hardlink の検証だけはテスト時に一時ディレクトリで作成します。

`unknown-image` は完全なイメージ参照でない短縮 ID の拒否を確認します。単体テスト内で定義を変形するテストには、正常系 fixture のメモリ上のコピーも使用します。
