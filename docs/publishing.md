# 課題の登録と公開

このリポジトリで課題定義を検証し、sandboxイメージをGHCRへ、解決済み課題JSONをmainの `release/` へ公開する。GitHub Releaseは使わない。実データとして `ex1/`、ビルド定義として `sandbox/` と `sandbox-runner/` を登録している。旧リポジトリの公開履歴・イメージlockは移行しない。

## 課題の登録

専用ディレクトリに [resource.yaml](resource.md) と参照素材を置き、ルートの `manifest.yaml` に登録する。

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

登録の `id` と `resource.id` は一致させる。`path` は `resource.yaml` を含むディレクトリ。課題内の `sandbox-image` には `ghcr.io/dsa-uts/dsa-resource-spec-sandbox-default:latest` のような完全なタグ参照、またはdigest参照を記述する。manifestのイメージIDを課題から参照する旧形式は使わない。

`context` と `dockerfile` はmanifestルートからの相対パス。リポジトリ内の `.`、`..`、相対symlinkを許可するが、範囲外への参照・絶対symlinkは禁止する。contextはディレクトリ、Dockerfileは通常ファイルでなければならない。対応platformは `linux/amd64` と `linux/arm64`。

`resource-ci build-images` では `image` に重複しない、タグ・digestなしのGHCR repositoryを指定する。移行元との `latest` 競合を避けるため、このリポジトリでは `dsa-resource-spec-sandbox-default` / `dsa-resource-spec-sandbox-runner` を使う。

## PRでの検証

```sh
go build -o /tmp/resource-ci ./cmd/resource-ci
/tmp/resource-ci check
```

CIは現在の課題定義・参照素材と、公開index・JSONの形式や整合性を検証する。`resource-ci` は `LoadManifest` を直接呼び出し、YAML解析・素材の読み込み・課題の検証をライブラリと共有する。

同じversionのまま課題定義や素材を変更してもよい。過去のコミットとの比較や、resource-hashの照合は行わない。公開済みのversionはスキップするため、変更を公開するには `resource.yaml` の `resource.version` を未公開の値に更新する。初回登録時も、そのversionのJSONを追加する。versionの増加順序は検査しない。

## mainでの公開順序

[Resources workflow](../.github/workflows/resources.yml) はmainへのpush、またはmainを指定した手動実行で以下を行う。PRからのpush権限は与えない。

1. 対象コミットを検証する。
2. 登録された全sandboxを毎回Buildxでビルドする。イメージ別のGitHub Actionsキャッシュを使う。
3. OCI形式でローカルに出力し、manifest/indexのdigestをGHCRの `latest` と比較する。同じならpushを省略する。
4. digestが変わった場合、`YYYYMMDDHHMMSS-sha256-<64桁のdigest>` の固定タグでpushし、その後 `latest` を更新する。日時はUTC。固定タグは86文字。既存タグの認証・通信エラーは「イメージなし」と扱わず失敗する。
5. 全イメージの処理が成功した後、未公開versionの課題をGoライブラリで読み込んだ `Resource` からJSON化する。各タグをレジストリで解決し、`repository@sha256:...` に置換する。同じタグは1回の公開処理で一度だけ解決する。明示済みのdigestはそのまま保持する。
6. 課題JSONとindexを同じコミットでmainへ追加する。

イメージ出力では `SOURCE_DATE_EPOCH=0` と `rewrite-timestamp=true` を使い、実行日時だけでdigestが変わらないようにする。provenance/SBOMの自動添付は無効にする。キャッシュの失効や外部パッケージの変更などによってdigestが変わることはある。ローカル出力には [BuildxのOCI exporter](https://docs.docker.com/build/exporters/oci-docker/) と、digestを保持してコピーする [regctl](https://github.com/regclient/regclient) を使う。

ビルド・push・タグ解決のいずれかが失敗すると、その実行では課題JSONを公開しない。複数イメージの途中で失敗した場合、先に成功したイメージのタグ更新は戻さない。

## 公開ファイル

```text
release/
  index.json
  ex1/
    v1.0.0.json
    v1.1.0.json
```

課題JSONは `Resource` の既存形式を維持し、素材も内包する。`DecodeResource` にそのまま渡せる。生成元のコミット情報はindexに記録する。

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

`path` はリポジトリルート基準。versionの列挙順には意味を持たせない。`resource-hash` はイメージタグをdigestに固定した公開対象の `Resource` を `json.Marshal` でJSON化し、そのバイト列のSHA-256を計算した記録で、公開判定には使わない。公開JSONを `DecodeResource` で読み込み、`Hash()` を呼ぶと同じ値を計算できる（整形されたJSONファイル自体のハッシュではない）。素材の内容やプリセットの実行可能フラグは含まれるが、YAMLのコメント・書式や参照元パスは含まれない。`source-commit` はCIが生成物を保存したコミットではなく、課題を生成したコミットを指す。

公開済みJSONは将来の `latest` 更新に追従しない。manifestから課題を取り除いても過去の公開ファイルは保持する。公開済み課題が参照するイメージ・固定タグもGHCRから削除しない運用とする。初期のindexは空で、初回公開はmainのCIで行う。

## 同時実行と再実行

workflow全体を同じconcurrency groupに置き、`queue: max` で直列化する。イメージ公開からタグ解決まで別実行が割り込まない。最大100件まで待機できるが、厳密なpush順は保証されない。独自のキューは設けない。[GitHubのconcurrency仕様](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency)

各実行はトリガー元のSHAをチェックアウトする。待機後に最新mainから課題を生成することはしない。保存時には別の一時worktreeに最新mainを取り、生成物だけを追加する。push時にmainが進んでいたら、同じ生成結果を使って最大5回やり直す。force pushはしない。公開順が前後しても、未公開のversionは追加する。

失敗時はActionsでその実行を手動再実行する。公開済みの同一versionは、ソースの変更有無にかかわらずスキップし、既存のJSONとindexエントリを保持する。未公開分は再実行時にビルドし、タグを解決し直す。JSONとindexは一括コミットなので、部分的なJSON公開は起きない。

生成コミットだけのpushは `paths-ignore: ['release/**']` でResources workflowの対象外にする。通常の `GITHUB_TOKEN` によるpushも後続workflowを起動しない。

## GitHub側の設定

公開jobは `GITHUB_TOKEN` の `contents: write` と `packages: write` を使う。mainのルールは、このCIによる生成コミットの直接pushを許可する必要がある。PR必須などの保護がbotのpushも禁止する場合、この方式では公開が失敗するのでリポジトリ側で許可を設定する。

GHCRのパッケージ可視性と利用側のpull権限を設定する。イメージへの書き込み権限もこのリポジトリに与える。これらのリモート設定はリポジトリ内のコードだけでは変更されない。

## ローカルの開発・検証

GoとGitが必要。実際のイメージ公開にはDocker Buildxとregctl v0.8.3も必要。

GitHub Actions 用の処理は `cmd/resource-ci` と `internal/publishing` に置く。汎用の `resource-spec` CLI と分離し、公開一覧の型・検証、Git操作、イメージ公開、課題公開を役割ごとのファイルにまとめている。

```sh
go build -o /tmp/resource-ci ./cmd/resource-ci
/tmp/resource-ci check --root .
# 以下はGHCRとmainへ実際に公開する操作。build-imagesの成功後にpublishする。
/tmp/resource-ci build-images --root .
/tmp/resource-ci publish --root .
```

`--root` は省略するとカレントディレクトリ。Actions 内では `build-images --gha-cache` でキャッシュを有効にする。`publish` は `origin/main` へ公開し、生成コミットにはGitHub Actions botの名前を使う。

```sh
go vet ./...
go test ./...
```

公開処理のテストも `go test ./...` で実行する。実際のGoライブラリ・一時Gitリポジトリ・ローカルbare remoteを使って、同一versionの変更を許容すること、公開時に既存JSONを保持すること、再実行、連続するversionの公開、push競合を検証する。Docker・レジストリ操作は状態を持つテスト用のコマンド実行処理で代替するため、GHCRへの書き込みは行わない。
