# 課題の公開

このリポジトリで課題の登録、イメージのビルド、GitHub Release への公開を行う。初期状態の `resources.yaml` は空で、公開対象はない。

## 1. 課題を登録する

専用ディレクトリに [resource.yaml](resource.md) と参照素材を置き、ルートの `resources.yaml` に登録する。

```yaml
resources:
  - id: sample
    path: exercises/sample/resource.yaml
sandbox-images: {}
```

一覧の `id` と定義の `resource.id` は一致させる。[testdata/valid](../testdata/valid) に素材を含む例があるが、イメージ名はプレースホルダーなので取得可能な参照に置き換える。

このリポジトリでイメージもビルドする場合は `sandbox-images` に追加する。

```yaml
sandbox-images:
  sandbox:
    build:
      context: images/sandbox
      dockerfile: images/sandbox/Dockerfile
      image: ghcr.io/your-org/sandbox
      platforms: [linux/amd64, linux/arm64]
```

`context` と `dockerfile` はリポジトリルートからの相対パス。`context` は専用のディレクトリ、`image` は書き込み可能なタグなし GHCR repository とする。対応する platform は上記の2種類。

ビルド入力は通常ファイルのみとし、`.git` と課題 manifest を含めない。Dockerfile は単一の tag+sha256 digest 固定 `FROM` を使う。`ADD` と外部 frontend は受け付けない。

## 2. ローカルで検証する

リポジトリルートで実行する。

```sh
go run ./cmd/resource-spec manifest .
```

課題一覧、各課題の定義・参照素材、イメージのビルド入力を検証し、結果を JSON で出力する。

### ZIP を手元で生成する

公開処理はイメージのタグを digest に解決し、`archive` に渡す。同じ処理を手元で試すには、`images.json` に元の参照と解決済み参照の対応表を用意する。

```json
{
  "ghcr.io/example/default:latest": "ghcr.io/example/default@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
```

上の digest は書式例。実際の配布ではレジストリで解決した値を使う。

```sh
go run ./cmd/resource-spec archive --images images.json --output /tmp/sample-v1.0.0.zip testdata/valid
```

出力先は課題ディレクトリの外を指定する。ZIP は直下の `resource.yaml` を配布用定義に置き換え、補助画像や空ディレクトリを含めて課題全体を収録し、実行ビットを保持する。作者用ファイルは変更しない。CLI は一時ファイルで生成し、成功した場合だけ出力先へ移動する。

## 3. 公開を設定する

リポジトリで release immutability を有効にし、Actions variable `IMMUTABLE_RELEASES_ENABLED=true` を設定する。この変数は設定済みという宣言であり、公開スクリプトが実際の設定を変更・照会するものではない。GHCR の可視性と利用側の pull 権限も設定する。

## 公開と再実行

[Resources workflow](../.github/workflows/resources.yml) はチェックアウトしたコードから CLI をビルドする。main への反映後、次の順に処理する。

1. イメージをキャッシュ付きでビルドし、`latest` を更新する。
2. 未公開課題のすべてのタグ参照を同じ repository の sha256 digest に固定する。既存 digest は保持する。未解決タグ、別 repository への置換、不正な digest は拒否する。
3. `<id>/<version>`（例: `sample/v1.0.0`）の draft Release に対象課題の ZIP を添付し、公開する。

公開する変更には、Release 一覧にある最新版より大きい `resource.version` を手動で指定する。版の飛び越しは可能。同一バージョンは再公開せず、古い版や build metadata だけを変えた版は拒否する。同じ版のまま編集しても公開済みの内容は変わらない。公開対象が空なら何も公開しない。

Release 作成前に main が進んでいたら中止する。失敗時に残った draft やタグは上書きしないため、内容と commit を確認して手動で対処する。公開済み Release・イメージの削除処理は設けない。
