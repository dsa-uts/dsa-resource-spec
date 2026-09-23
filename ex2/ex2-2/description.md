# 課題2-2 キューの実装 (基本課題)
[課題リンク](https://www.coins.tsukuba.ac.jp/~amagasa/lecture/dsa-jikken/report2/#2-2)

課題2-2で実装する以下のプログラムコードをアップロードせよ。

* `queue.h`
* `queue.c`
* `main_queue.c`
* `Makefile`

アップロードされたプログラムコードを用いて、以下の項目をチェックする

1. `Makefile`に記載されているターゲット`queue`をコンパイル・実行できること。
2. キューの実装において，以下の要件を満たすこと
    * キューに整数を1つ格納し，それが取り出せる
    * キューに整数を複数連続して格納し，それが格納した順番で取り出せる
    * キューに格納するデータが配列の末尾と先頭にまたがる場合で，上の 1, 2 が行える
    * キューに格納するデータが配列の末尾と先頭にまたがる場合と，そうでない場合の両方の状態について，次に示す状態はエラーとして扱う．このとき、 **標準エラー出力（stderr）** にエラーメッセージを表示し，`exit`関数を用いてプログラムを異常終了させる．
        1. 要素を取り出し，キューが空になった後にさらに要素取り出そうとした時
        2. キューが一杯の時にさらに格納しようとした時
        * プログラムが異常終了する場合は戻り値0以外で終了すること．例えば、`exit(EXIT_FAILURE)`を呼び出す．
        * エラーメッセージの文面は任意だが、発生した問題の意味がわかる文が望ましい

2.の要件については、こちらで用意した`test_queue.c`をコンパイル・実行してチェックする．

#### ソースコード test_queue.c
```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "queue.h"

int main(void) {
  int n, q;
  scanf("%d %d", &n, &q);

  Queue *que = create_queue(n);

  char op[15];

  while (q--) {
    scanf("%s", op);
    if (strcmp(op, "enq") == 0) {
      int x;
      scanf("%d", &x);
      enqueue(que, x);
    } else if (strcmp(op, "deq") == 0) {
      int x = dequeue(que);
      printf("pop: %d\n", x);
    } else {
      fprintf(stderr, "error: invalid operation\n");
      exit(EXIT_FAILURE);
    }
    display(que);
  }

  delete_queue(que);
}
```

このソースコードでは、キューの容量$n$とクエリの数$q$、そして、各クエリの内容$op_i$を以下の形式で受け取る。

> $n$ $q$\
> $op_1$\
> $op_2$\
> $\dots$\
> $op_q$

$op_i$は以下のいずれかである。
* $enq\,\,\,\,x$ : キューに整数$x$を格納する
* $deq$ : キューから整数を1つ取り出す。

#### 制約
* $1\leq n \leq 10$
* $1\leq q \leq 20$
* $1\leq x \leq 50$

#### 出力
* 正常時
    * 各クエリを処理する度に、キューの各要素を空白区切りで1行に出力する。
    * `deq`クエリでキューから整数$x$を取り出した場合は、`pop: x`というメッセージを末尾に改行をつけて出力する。
    * 全てのクエリを処理した後、戻り値0で正常終了する。
* 異常時
    * エラーメッセージを標準エラー出力に出力する。エラーの内容は任意だが、発生した問題の意味がわかる文が望ましい。
    * プログラムを戻り値0以外で終了させる。例えば、`exit(EXIT_FAILURE)`を呼び出す。

#### 具体例
##### 入力1
```text
5 6
enq 1
enq 2
enq 3
deq
deq
enq 4
```

##### 出力1
```text
1 
1 2 
1 2 3 
pop: 1
2 3 
pop: 2
3 
3 4

```

##### 入力2
```text
5 10
enq 1
enq 3
enq 5
enq 7
enq 11
deq
deq
deq
deq
deq

```

##### 出力2
```text
1 
1 3 
1 3 5 
1 3 5 7 
1 3 5 7 11 
pop: 1
3 5 7 11 
pop: 3
5 7 11 
pop: 5
7 11 
pop: 7
11 
pop: 11

```

##### 入力3 (異常終了の例)
```text
3 5
enq 1
enq 2
enq 3
enq 4
deq
```

##### 出力3 (異常終了の例)
```text
1 
1 2 
1 2 3 
キューが一杯です
```
(戻り値0以外で終了)

## 提出方法

基本課題・発展課題をまとめて、次のファイルを提出ルート直下に配置して提出せよ。
小問ごとのフォルダには分けない。1つの提出に各小問のワークフローを実行する。
発展課題を提出する場合は、そのソースコードも同じ場所に配置する。

- `linked_list.h`、`linked_list.c`、`main_linked_list.c`
- `queue.h`、`queue.c`、`main_queue.c`
- `doublylinked_list.h`、`doublylinked_list.c`、`main_doublylinked_list.c`（発展課題）
- 共通の `Makefile`

共通の Makefile には、提出する各小問のターゲットを定義する。

```make
CC = gcc
.PHONY: all
all: linked_list queue doublylinked_list

linked_list: linked_list.o main_linked_list.o
	$(CC) $(LDFLAGS) -o $@ $^ $(LDLIBS)

queue: queue.o main_queue.o
	$(CC) $(LDFLAGS) -o $@ $^ $(LDLIBS)

doublylinked_list: doublylinked_list.o main_doublylinked_list.o
	$(CC) $(LDFLAGS) -o $@ $^ $(LDLIBS)
```

採点用のテストコードと入力ファイルは採点側が用意するため、提出不要。
テストコードは `/preset/ex2-N/`（`N` は小問番号）から、提出されたヘッダーを
`gcc -I.` で参照し、対応するオブジェクトファイルとリンクしてコンパイルする。
