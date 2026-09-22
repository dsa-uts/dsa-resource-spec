# 課題2-1 連結リストの実装 (基本課題)
[課題リンク](https://www.coins.tsukuba.ac.jp/~amagasa/lecture/dsa-jikken/report2/#2-1)

課題2-1で実装する以下のプログラムコードをアップロードせよ．

* `linked_list.h`
* `linked_list.c`
* `main_linked_list.c`
* `Makefile`

アップロードされたプログラムコードを用いて，以下の項目をチェックする

1. `Makefile`に記載されているターゲット`linked_list`をコンパイル・実行できること。
2. 連結リストの実装において，以下の要件を満たすこと
    * 先頭にセルを挿入できる
    * 指定したセルの直後にセルを挿入できる
    * 先頭のセルを削除できる
    * 指定したセルの直後のセルを削除できる
    * リストの要素を順に標準出力に表示できる

2.の要件については、こちらで用意した`test_linked_list.c`をコンパイル・実行してチェックする．

#### テストファイル test_linked_list.c
```c
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "linked_list.h"

int main(void) {
  int q;

  char op[15];

  scanf("%d", &q);

  while (q--) {
    scanf("%s", op);
    if (strcmp(op, "insert") == 0) {
      int pos, x;
      scanf("%d %d", &pos, &x);
      pos -= 1;
      assert(pos >= 0);
      if (pos == 0) {
        insert_cell_top(x);
        goto DISPLAY;
      }
      Cell* c = head;
      int i;
      for (i = 0; i < pos - 1; ++i) {
        c = c->next;
      }
      insert_cell(c, x);
    } else if (strcmp(op, "delete") == 0) {
      int pos;
      scanf("%d", &pos);
      pos -= 1;
      assert(pos >= 0);
      if (pos == 0) {
        delete_cell_top();
        goto DISPLAY;
      }
      Cell* c = head;
      int i;
      for (i = 0; i < pos - 1; ++i) {
        c = c->next;
      }
      delete_cell(c);
    } else {
      fprintf(stderr, "invalid operation: %s\n", op);
      exit(EXIT_FAILURE);
    }

  DISPLAY:
    display();
  }

  return 0;
}

```

このソースコードでは、クエリの数$Q$とクエリの内容$op_i$を以下の形式で受け取る。
<div style="background-color: #f0f0f0; padding: 10px; border-radius: 5px; color: black;">

$Q$ <br>
$op_1$ <br>
$op_2$ <br>
$\dots$ <br>
$op_Q$
</div>

$op_i$は以下のいずれかである。

* $insert\,\,\,\,pos\,\,\,\,x$ : 先頭から$pos$番目のセルの位置に$x$を挿入する。元々$pos$番目のセルは、新しいセルの前にくる。
* $delete\,\,\,\,pos$ : 先頭から$pos$番目のセルを削除する

#### 制約
* $1\leq Q \leq 15$
* $1\leq pos$
* $1\leq x \leq 50$

#### 出力
* 正常時
    * $i$行目に、$i$番目のクエリを処理した後のリストの要素を空白区切りで1行に出力する。
    * 全てのクエリを処理した後、戻り値0で正常終了する
* 異常時
    * 正しく実装できていれば正常終了するようなテストケースのみを与えるので、エラー処理の実装は任意とする。

#### 具体例
##### 入力1
```text
4
insert 1 1
insert 2 3
insert 2 2
delete 3
```
##### 出力1
```text
1
1 3
1 2 3
1 2
```

##### 入力2
```text
10
insert 1 1
insert 2 5
insert 3 7
insert 4 8
insert 5 10
insert 2 2
insert 3 3
insert 4 4
insert 6 6
insert 9 9
```
##### 出力2
```text
1 
1 5 
1 5 7 
1 5 7 8 
1 5 7 8 10 
1 2 5 7 8 10 
1 2 3 5 7 8 10 
1 2 3 4 5 7 8 10 
1 2 3 4 5 6 7 8 10 
1 2 3 4 5 6 7 8 9 10 
```

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
