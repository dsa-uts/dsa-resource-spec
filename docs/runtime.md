# Backend・Judge の実行規則

[課題定義](resource.md) を実行するシステムの契約。この Go モジュールは定義の検証・素材の読み込みと出力比較関数 `MatchOutput` を提供する。以下の実行、採点、権限制御、提出物の正規化は Backend・Judge の責務とする。

## 実行権限と順序

Validation Request は `public` Job のみ実行する。`private` Job は Manager / Admin の Request でのみ実行でき、結果と成果物も Manager / Admin だけが参照できる。一般 User には stdout・stderr・status を含めて表示しない。

Judge は `depends` の依存関係を満たした Job を一つずつ実行する。依存関係のない Job 間の実行順に意味は持たせない。

## Sandbox の固定設定

sandbox は gVisor (`runsc`) RuntimeClass を指定した Kubernetes Pod として専用 Namespace で実行する。以下は platform 固定で、課題定義からは変更できない。

- 外向き通信は既定で拒否する。
- Linux capabilities はすべて削除し、`no_new_privileges` を適用する。
- 固定の非 root ユーザーで Step を実行する。
- root filesystem は読み取り専用とし、書き込み先は作業領域など許可した場所に限定する。
- `/bin/bash` を提供し、監査ログを必須とする。

CPU・メモリ・プロセス数・実行時間・出力サイズなどの上限は Job の `limits` に従う。

## ファイルの配置と回収

Job ごとに独立した作業領域を作成し、次の順で処理する。同じ Job の Step 間では作業領域を共有し、途中で消去しない。

1. 提出ファイルと入力成果物を、作業領域に書き込み可能な通常ファイルとしてコピーする。
2. Preset を読み取り専用の `/preset` に配置する。
3. Step を順に実行する。
4. 実行結果と、存在する出力成果物を回収する。
5. 作業領域を削除する。

提出ファイルはコピーなので、実行中の変更は元の提出物に反映しない。解決済みの `Step.Stdin` を Judge が標準入力に流し、`Expected.Stdout` / `Stderr` の `Content` を Judge 内で比較に使う。これらのファイルは作業領域へ配置しない。

Preset は `Executable` が true なら実行できるようにし、実行ユーザーによる変更・削除・置換を禁止する。親ディレクトリへの書き込み権限でも置換できるため、ファイル単体の権限変更だけでは不十分。

### 成果物の回収

- 入出力成果物のパスは Job の作業領域を基準に解釈し、Step 内の `cd` の影響を受けない。
- 出力は全 Step の実行後、sandbox の削除前に回収する。ファイルがない場合は回収状況を実行結果に記録する。
- 入力成果物がない場合は、sandbox 開始前に setup failure とする。
- setup failure・timeout・Step failure・成果物回収失敗のいずれでも実行結果を保存する。
- `limits.artifact-size` を超えるファイルは保存せず、回収状況を記録する。
- 実行ビットだけを維持し、実行可能なら `0755`、それ以外は `0644` にする。owner・group・suid・sgid・sticky bit は維持しない。
- 成果物は通常ファイルのみ。ディレクトリを指定すると、出力では回収失敗、入力では setup failure とする。
- 提出ファイル・成果物の symlink、hardlink、device、FIFO、socket は検証エラーとする。Preset は解決済みの bytes から通常ファイルとして配置する。作者の素材参照に含まれる symlink は読み込み時に解決済み。
- 成果物は信頼できない入力として扱い、Judge のホスト上では実行しない。

## Step の実行

作業領域の絶対パスは Judge が決める。各 Step は Job の作業領域をカレントディレクトリとして開始し、Step 内の `cd` は次の Step に引き継がない。Preset を作業領域へコピーする場合は `cp /preset/sample.txt .` のように記述できる。

Judge は `/bin/bash -e -o pipefail -c <run文字列>` を実行する。`run` は単一の引数としてそのまま渡し、Judge 側で分割・展開しない。Bash が展開・パイプ・リダイレクト・glob を解釈し、コマンドを sandbox の `PATH` で解決する。絶対パス・相対パスによるコマンド指定も許可する。

`-e` により通常のコマンドの失敗で停止し、`pipefail` によりパイプラインは右端の非ゼロ終了コードを返す。`if`・`&&`・`||` などには Bash の `-e` の例外規則が適用される。Step の終了コードは Bash の終了コードとし、作業ディレクトリ・入出力・timeout はスクリプト全体に適用する。

```yaml
run: |
  make
  ./main < input.txt | sort > result.txt
```

この例の `input.txt` は作業領域のファイル。`make` が失敗すると後続行は実行しない。パイプライン内の失敗も Step の非ゼロ終了コードになる。

## タイムアウト

YAML の `step.timeout`、省略時は `job.limits.step-timeout` を読み込み時に解決する。Judge は解決済み `Step.Timeout` を使う。Job 全体は各 Step の実効 timeout の合計に Judge 内部の buffer（既定10秒）を加える。たとえば 30秒・10秒・10秒の Step なら60秒になる。

課題定義から変更できるのは Step の時間制限だけ。`job.limits.timeout-seconds` と `job.limits.timeout-buffer-seconds` は受け付けない。

## 出力の比較

stdout / stderr は UTF-8 テキストとして扱う。出力・期待値のいずれかがデコードできない場合は比較失敗とする。`MatchOutput` はこの場合 `false, nil` を返す。未知・空のモードはエラーとし、期待値の省略による比較のスキップは呼び出し側が判断する。

| `match` | 比較方法 |
| --- | --- |
| `exact` | 正規化せず完全一致。末尾の改行も比較する。 |
| `easy` | 以下の正規化後、行数・各行の要素数・要素の順序と内容が一致すること。 |
| `sorted` | `easy` の正規化後、各行の要素を文字列の辞書順で昇順ソートして比較する。行順は変えず、重複する要素の個数も比較する。 |

`easy` と `sorted` の正規化は出力と期待値の両方に次の順で適用する。

1. 全体の末尾から改行を一つだけ取り除く。対象は `\n`・`\r\n`・`\r`。
2. `\n`・`\r\n`・`\r` を区切りとして行に分割する。CRLF は一つの区切りとし、空行は保持する。その他の Unicode whitespace は行区切りにしない。
3. 各行の先頭・末尾の Unicode whitespace を取り除く。
4. 1 文字以上の Unicode whitespace で各行を分割し、要素列にする。

例: `easy` では `"  A\tB  \nC　D\n"` と `"A B\nC D"` が一致する。`sorted` では `"B A\n3 2 1"` と `"A B\n1 2 3"` も一致する。

## 提出アーカイブの正規化

提出物は受講者の環境差を吸収するため、配置前にパスを正規の POSIX 相対パスへ変換する。提出物の同一性を判定する content hash は、正規化後のファイルツリーから計算する。

- `\` は区切り文字として `/` に変換してよい。
- 正規化後の空パス、`.`、絶対パス、`..` の階層、NUL は拒否する。
- Windows drive path (`C:\...`) と UNC path (`\\server\share\...`) は拒否する。
- 正規化後の同じパスに複数の entry が衝突する場合は拒否する。
- symlink、hardlink、device、FIFO、socket は拒否する。
- ホストの filesystem に直接展開しない。メモリ上または sandbox 用一時領域で正規化と検証を終えてから、作業領域にコピーする。
