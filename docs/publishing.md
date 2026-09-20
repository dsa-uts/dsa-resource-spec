# 課題の登録とイメージのビルド

このリポジトリで課題の登録・検証と、GHCR へのイメージのビルド・push を行う。初期状態の `manifest.yaml` は空で、登録された課題やビルド対象のイメージはない。解決済み課題は `inspect` で JSON に出力できる。ホスティング先や自動公開の運用は未定。

## 1. 課題を登録する

専用ディレクトリに [resource.yaml](resource.md) と参照素材を置き、ルートの `manifest.yaml` に登録する。

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

`context` と `dockerfile` はリポジトリルートからの相対パス。`context` はディレクトリ（ルートの `.` も可）。両パスとも `./` と範囲内の `..` を許可し、絶対パス・backslash・colon・NUL・空文字は禁止する。`image` は空でない文字列とし、レジストリ・タグの形式や重複の可否は、利用するビルド処理が決める。対応する platform は上記の2種類。

同梱の Actions は GHCR にログインし、ビルドスクリプトは `image` に `:latest` を付けて push する。この Actions を使う場合は、`image` に書き込み可能なタグなし GHCR repository を指定する。

Dockerfile とビルドコンテキストは、manifest の読み込み時点でリポジトリ内に存在することを必須とする。manifest の検証では、`context` がディレクトリ、`dockerfile` が通常ファイルであることを確認する。範囲内を指す相対 symlink は許可するが、マニフェストのディレクトリ外への参照と絶対 symlink は拒否する。内容は読み込まず、レビューと Docker によるビルドで確認する。

## 2. ローカルで検証する

リポジトリルートで実行する。

```sh
go run ./cmd/resource-spec validate .
```

課題一覧、各課題の定義・参照素材、イメージのビルド設定を検証する。

解決済みのマニフェスト全体は `resource-spec manifest .`、一つの課題は次で出力する。

```sh
resource-spec inspect . sample > sample.json
```

`sample.json` は `DecodeResource` にそのまま渡せる。元の素材ファイルを添付する必要はない。[JSON の契約](resolved-resource.md) を参照。

## 3. イメージのビルドを設定する

GHCR の可視性と利用側の pull 権限を設定する。

[Resources workflow](../.github/workflows/resources.yml) はチェックアウトしたコードから CLI をビルドし、`validate` で課題一覧を検証する。main への反映後、または main に対する手動実行で、`sandbox-images` のイメージをキャッシュ付きでビルドして GHCR に push し、`latest` を更新する。ビルド対象が空ならイメージのビルド・push は行わない。
