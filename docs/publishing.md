# 課題の登録とイメージのビルド

このリポジトリで課題の登録・検証と、GHCR へのイメージのビルド・push を行う。初期状態の `resources.yaml` は空で、登録された課題やビルド対象のイメージはない。課題の配布方法は未定。

## 1. 課題を登録する

専用ディレクトリに [resource.yaml](resource.md) と参照素材を置き、ルートの `resources.yaml` に登録する。

```yaml
resources:
  - id: sample
    path: exercises/sample # resourceのコンテキストディレクトリ。トップにresource.yamlが置かれている。
sandbox-images: {}
```

一覧の `id` と定義の `resource.id` は一致させる。[testdata/resource/valid/basic](../testdata/resource/valid/basic) に素材を含む例があるが、イメージ名はプレースホルダーなので取得可能な参照に置き換える。

このリポジトリでイメージもビルドする場合は `sandbox-images` に追加する。

```yaml
sandbox-images:
  sandbox:
    context: images/sandbox
    dockerfile: images/sandbox/Dockerfile
    image: ghcr.io/your-org/sandbox
    platforms: [linux/amd64, linux/arm64]
```

`context` と `dockerfile` はリポジトリルートからの相対パス。`context` は専用のディレクトリ、`image` は書き込み可能なタグなし GHCR repository とする。対応する platform は上記の2種類。

Dockerfile とビルドコンテキストの内容はレビューと Docker によるビルドで確認する。manifest の検証ではビルド入力の存在や内容は確認しない。

## 2. ローカルで検証する

リポジトリルートで実行する。

```sh
go run ./cmd/resource-spec manifest .
```

課題一覧、各課題の定義・参照素材、イメージのビルド設定を検証し、結果を JSON で出力する。

## 3. イメージのビルドを設定する

GHCR の可視性と利用側の pull 権限を設定する。

[Resources workflow](../.github/workflows/resources.yml) はチェックアウトしたコードから CLI をビルドし、`manifest` で課題一覧を検証する。main への反映後、または main に対する手動実行で、`sandbox-images` のイメージをキャッシュ付きでビルドして GHCR に push し、`latest` を更新する。ビルド対象が空ならイメージのビルド・push は行わない。
