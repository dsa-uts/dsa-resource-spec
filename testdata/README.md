# CLI fixture

`go test ./...` で CLI を一度ビルドし、`cli/valid/<case>` と `cli/invalid/<case>` を自動探索して実行します。ケースを追加する際、Go のテスト一覧への登録は不要です。

```text
cli/valid/basic/
  input/
    manifest.yaml
    sample/
      resource.yaml
      description.md
      expected.txt
  want/
    catalog.json
    show/
      sample.json
cli/invalid/missing-expected/
  input/
    manifest.yaml
    sample/resource.yaml
```

- `input/` はそのまま CLI に渡せるマニフェストのルートです。
- 正常系は `validate` の終了コード0と空の stdout、`catalog` と `show` の終了コード0および期待JSONを検証します。`want/catalog.json` に載る全IDについて `want/show/<id>.json` が必要です。空のカタログでは show の期待値は不要です。
- JSONの空白とオブジェクトのキー順は無視します。配列順、数値（大きな整数も含む）、フィールドの有無、`null` と空配列は区別します。
- 異常系は `validate` の終了コードが非0なら成功です。stderr の有無や文言、具体的な終了コードは比較しません。起動失敗・タイムアウト・シグナル終了はテスト失敗です。
- 引数の誤りと、`catalog`・`show` への入力エラーの伝播は代表ケースで別途検証します。

## 追加・更新

正常系は近いケースをコピーして入力と期待JSONを編集します。異常系は原則として意図した不備を一つだけ含めます。複数の不備があると、別の理由による失敗でもテストが通ってしまいます。

```sh
# 入力を直接確認
go run ./cmd/resource-spec validate testdata/cli/valid/basic/input
go run ./cmd/resource-spec show testdata/cli/valid/basic/input sample

# 期待JSONを明示的に再生成する例（成功を確認してから置き換える）
go run ./cmd/resource-spec catalog testdata/cli/valid/basic/input > /tmp/catalog.json &&
  python3 -m json.tool /tmp/catalog.json > testdata/cli/valid/basic/want/catalog.json
go run ./cmd/resource-spec show testdata/cli/valid/basic/input sample > /tmp/sample.json &&
  python3 -m json.tool /tmp/sample.json > testdata/cli/valid/basic/want/show/sample.json

# 一つのケースを検証
go test ./cmd/resource-spec -run '^TestCLI$/^valid$/^basic$' -v
```

生成結果は仕様と照合し、差分をレビューしてください。通常のテスト実行では期待値を書き換えません。

## ケースの役割

- `basic`: 既定値、単位変換、素材の展開、依存関係、artifact、未指定の期待値。
- `multiple-resources`: manifest の登録順と全リソースの出力。
- `explicit-values`: 明示した制限値、stdin、timeoutの上書き、終了コード0と255。
- `shared-materials`: 共有素材、バイナリ、空文字列、実行ビット、symlink と親ディレクトリ、未使用の壊れたリンク。
- `yaml-anchors`: YAMLアンカーと既定のvisibility。
- `*symlink`、`build-*`: リンク解決とビルド設定のパス検証。
- `external-*`: `input/` の外に実在するファイルへの相対参照を拒否。ケース内の `outside/` は拒否対象です。

symlink と `shared-materials/input/shared/tool` の実行ビットを checkout 時にも保持してください。絶対参照だけは一時ディレクトリで準備してCLIを実行します。

`DecodeResource` はCLIから呼ばれない公開APIなので、ライブラリ側に不正JSONの拒否テストと、正常系の期待JSONを使う往復テストを残しています。manifest の読み込みテストはCLI側に集約しています。
