# 設計の理由

操作方法は [README](../README.md)、書式は [Resource 仕様](resource.md)、配布の手順は [公開手順](publishing.md) を参照。

## 課題ごとにバージョンを管理する

課題のversionは、生成元コミットにおける定義・参照素材と、公開時点で解決したsandboxイメージのdigestを固定したスナップショットである。mainの `release/<課題ID>/<version>.json` に保存し、公開後は変更・削除しない。

sandboxの `latest` 更新は既存の課題に反映しない。新しい環境を適用する課題だけversionを上げる。同じコミットでsandboxと課題を更新した場合は、イメージのビルド・pushが成功してから課題内のタグを解決する。

公開一覧と生成元コミットSHAは `release/index.json` に記録し、課題JSONと同時にコミットする。JSON自体は実行に必要なResourceの形式を維持する。詳細は [公開手順](publishing.md) を参照。

## その他

入力は呼び出し中に変更しないことを前提とする。
hardlink は通常ファイルとして読み、inode の共有まではチェックしない。

