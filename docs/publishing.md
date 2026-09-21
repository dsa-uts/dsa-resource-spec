# 課題の登録と公開

このリポジトリで課題定義を検証し、サンドボックスイメージを GHCR へ、解決済み課題 JSON を main の `release/` へ公開する。GitHub Release は使わない。実データとして `ex1/`、ビルド定義として `sandbox/` と `sandbox-runner/` を登録している。旧リポジトリの公開履歴・イメージの固定情報は移行しない。

## 課題の登録

専用ディレクトリに [resource.yaml](resource.md) と参照ファイルを置き、ルートの `manifest.yaml` に登録する。

```yaml
resources:
  - id: ex1
    path: ex1
sandbox-images:
  default:
    context: sandbox
    dockerfile: sandbox/Dockerfile
    image: ghcr.io/dsa-uts/dsa-resource-spec-sandbox-default
    platforms: [linux/amd64]
```

登録の `id` と `resource.id` は一致させる。`path` は `resource.yaml` を含むディレクトリ。課題内の `sandbox-image` には `ghcr.io/dsa-uts/dsa-resource-spec-sandbox-default:latest` のような完全なタグ参照、またはダイジェスト参照を記述する。マニフェストのイメージ ID を課題から参照する旧形式は使わない。

`context` と `dockerfile` はマニフェストを置いたディレクトリからの相対パス。リポジトリ内の `.`、`..`、相対パスのシンボリックリンクを許可するが、範囲外への参照・絶対パスのシンボリックリンクは禁止する。`context` はディレクトリ、`dockerfile` は通常ファイルでなければならない。対応プラットフォームは `linux/amd64` と `linux/arm64`。

`resource-ci build-images` では `image` に重複しない、タグ・ダイジェストを含まない GHCR リポジトリ名を指定する。移行元との `latest` 競合を避けるため、このリポジトリでは `dsa-resource-spec-sandbox-default` / `dsa-resource-spec-sandbox-runner` を使う。

## PR での検証

```sh
go build -o /tmp/resource-ci ./cmd/resource-ci
/tmp/resource-ci check
```

CI は現在の課題定義・参照ファイルと、公開一覧・JSON の形式や整合性を検証する。`resource-ci` は `LoadManifest` を直接呼び出し、YAML 解析・参照ファイルの読み込み・課題の検証をライブラリと共有する。

新バージョンを公開するには `resource.yaml` の `resource.version` を未公開の値に更新する。初回登録時も、そのバージョンの JSON を追加する。

## main での公開順序

[Resources workflow](../.github/workflows/resources.yml) は main への push、または main を指定した手動実行で以下を行う。PR からの push 権限は与えない。

1. 対象コミットを検証する。
2. 登録されたすべてのサンドボックスイメージを毎回 Buildx でビルドする。イメージ別の GitHub Actions キャッシュを使う。
3. OCI 形式でローカルに出力し、マニフェストまたはインデックスのダイジェストを GHCR の `latest` と比較する。同じなら push を省略する。
4. ダイジェストが変わった場合、`YYYYMMDDHHMMSS-sha256-<64桁のdigest>` の固定タグで push し、その後 `latest` を更新する。日時は UTC。固定タグは 86 文字。既存タグの確認時に認証・通信エラーが発生した場合は、「イメージなし」と扱わず処理を失敗させる。
5. 全イメージの処理が成功した後、未公開バージョンの課題を Go ライブラリで読み込んだ `Resource` から JSON 化する。各タグをレジストリで解決し、`repository@sha256:...` に置換する。同じタグは 1 回の公開処理で一度だけ解決する。明示済みのダイジェストはそのまま保持する。
6. 課題 JSON と公開一覧を同じコミットで main へ追加する。

ビルド・push・タグ解決のいずれかが失敗すると、その実行では課題 JSON を公開しない。複数イメージの途中で失敗した場合、先に成功したイメージのタグ更新は戻さない。

## 公開ファイル

```text
release/
  index.json
  ex1/
    v1.0.0.json
    v1.1.0.json
```

課題 JSON は `Resource` の既存形式を維持し、参照ファイルの内容も含む。`DecodeResource` にそのまま渡せる。生成元のコミット情報は公開一覧に記録する。

```json
{
  "resources": {
    "ex1": {
      "v1.0.0": {
        "path": "release/ex1/v1.0.0.json",
        "source-commit": "<生成元コミットの完全なSHA>",
        "resource-hash": "sha256:<イメージタグ固定後のResourceのJSONのハッシュ>"
      }
    }
  }
}
```

`path` はリポジトリルート基準。

## GitHub 側の設定

公開ジョブは `GITHUB_TOKEN` の `contents: write` と `packages: write` を使う。main ブランチのルールは、この CI による生成コミットの直接 push を許可する必要がある。

## ローカルの開発・検証

Go と Git が必要。実際のイメージ公開には Docker Buildx と regctl v0.8.3 も必要。

GitHub Actions 用の処理は `cmd/resource-ci` と `internal/publishing` に置く。汎用の `resource-spec` CLI と分離し、公開一覧の型・検証、Git 操作、イメージ公開、課題公開を役割ごとのファイルにまとめている。

```sh
go build -o /tmp/resource-ci ./cmd/resource-ci
/tmp/resource-ci check --root .
# 以下はGHCRとmainへ実際に公開する操作。build-imagesの成功後にpublishする。
/tmp/resource-ci build-images --root .
/tmp/resource-ci publish --root .
```

`--root` は省略するとカレントディレクトリ。Actions 内では `build-images --gha-cache` でキャッシュを有効にする。`publish` は `origin/main` へ公開し、生成コミットには GitHub Actions bot の名前を使う。

```sh
go vet ./...
go test ./...
```

