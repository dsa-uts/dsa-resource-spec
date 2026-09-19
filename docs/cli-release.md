# CLI のリリース

[CLI Release workflow](../.github/workflows/cli-release.yml) は `cli/v*` タグの push で Linux 用バイナリ（amd64・arm64）と `SHA256SUMS` を GitHub Releases に公開する。

## 公開する

ワークフローを含む公開対象のコミットで実行する。

```sh
git tag cli/v1.0.0
git push origin cli/v1.0.0
```

`cli/v1.1.0-rc.1` などは Pre-release として公開する。タグの衝突を避けるため、課題 ID に `cli` は使わない。

既存 Release は上書きしない。失敗して draft が残った場合は、内容を確認して削除してから再実行する。

## ダウンロードして使う

[Releases](https://github.com/dsa-uts/dsa-resource-spec/releases) から取得する。GitHub CLI を使う場合：

```sh
# arm64 の場合は amd64 を arm64 に置き換える
gh release download cli/v1.0.0 --repo dsa-uts/dsa-resource-spec \
  --pattern resource-spec-linux-amd64 --pattern SHA256SUMS
sha256sum --check --ignore-missing SHA256SUMS
chmod +x resource-spec-linux-amd64
./resource-spec-linux-amd64 validate path/to/resource-directory
```
