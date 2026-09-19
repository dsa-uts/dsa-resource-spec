# 検証 fixture

`valid` は参照素材を含む単独課題です。`invalid/*/sample/resource.yaml` は異常系の定義で、正常系の定義と差し替えて検証します。

`id-mismatch` と `duplicate-yaml-key/resources.yaml` は manifest 検証で扱います。`unknown-image` は完全なイメージ参照でない短縮 ID の拒否を確認します。
