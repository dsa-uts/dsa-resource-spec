# 課題2-3 双方向循環リストの実装 (発展課題)
[課題リンク](https://www.coins.tsukuba.ac.jp/~amagasa/lecture/dsa-jikken/report2/#2-3)

課題2-3で実装する以下のソースコードをアップロードせよ。

* `doublylinked_list.h`
* `doublylinked_list.c`
* `main_doublylinked_list.c`
* `Makefile`

アップロードされたソースコードを用いて、以下の項目をチェックする

1. `Makefile`に記載されているターゲット`doublylinked_list`をコンパイル・実行できること。
2. 双方向循環リストの実装において、以下の要件を満たすこと
    * セルを、リストの先頭、末尾、中間に挿入できること
    * 先頭セル、末尾のセル、中間のセルが削除できること
    * `SearchCell()`関数によって、先頭セル、末尾のセル、中間のセルを探せること
3. 双方向循環リストの内容を配列やファイルに書き出したり、そこから読み込んだりする関数の実装において、以下の要件を満たすこと
    * `ReadFromArray()`関数により、配列の内容をリストに追加できること
    * `WriteToArray()`関数により、リストの内容を配列に書き出せること
    * `WriteToFile()`関数により、リストの内容をファイルに出力できること
    * `ReadFromFile()`関数により、`WriteToFile()`関数で出力したファイルからリストを読み込み直せること

2.と3.の要件については、こちらで用意した`test_doublylinked_list.c`と`test_util.c`をコンパイル・実行してチェックする

## 双方向循環リストの実装チェック
#### ソースコード test_doublylinked_list.c
```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <assert.h>
#include "doublylinked_list.h"

int main(void) {
  int q;
  scanf("%d", &q);
  char op[16];
  char dir[15];

  Cell* head = CreateCell(0, true);

  while (q--) {
    scanf("%15s", op);
    if (strcmp(op, "insert") == 0) {
      int pos, val;
      scanf("%s %d %d", dir, &pos, &val);
      assert(pos > 0);
      pos -= 1;
      Cell* p = head;
      if (strcmp(dir, "prev") == 0) {
        int i;
        for (i = 0; i < pos; ++i) {
          p = p->prev;
          if (p == head)
            p = p->prev;
        }
        InsertPrev(p, CreateCell(val, false));
      } else if (strcmp(dir, "next") == 0) {
        int i;
        for (i = 0; i < pos; ++i) {
          p = p->next;
          if (p == head)
            p = p->next;
        }
        InsertNext(p, CreateCell(val, false));
      } else {
        printf("error: invalid direction for insert\n");
        exit(EXIT_FAILURE);
      }
    } else if (strcmp(op, "delete") == 0) {
      int pos;
      scanf("%s %d", dir, &pos);
      assert(pos > 0);
      Cell *p = head;
      if (strcmp(dir, "prev") == 0) {
        int i;
        for (i = 0; i < pos; ++i) {
          p = p->prev;
          if (p == head)
            p = p->prev;
        }
        DeleteCell(p);
      } else if (strcmp(dir, "next") == 0) {
        int i;
        for (i = 0; i < pos; ++i) {
          p = p->next;
          if (p == head)
            p = p->next;
        }
        DeleteCell(p);
      } else {
        printf("error: invalid direction for delete\n");
        exit(EXIT_FAILURE);
      }
    } else if (strcmp(op, "display") == 0) {
      Display(head);
    } else if (strcmp(op, "display-reverse") == 0) {
      DisplayReverse(head->prev);
    } else if (strcmp(op, "search") == 0) {
      int val;
      scanf("%d", &val);
      Cell *p = SearchCell(head, val);
      if (p == NULL) {
        printf("QUERY: %d not found\n", val);
      } else {
        printf("QUERY: %d found\n", p->data);
      }
    } else {
      printf("error: invalid operation\n");
      exit(EXIT_FAILURE);
    }
  }

  // free all cells
  while (head->next != head) {
    DeleteCell(head->next);
  }
  DeleteCell(head);

  return 0;
}

```
このソースコードでは、クエリの数$Q$、クエリの内容$op_i$を以下の形式で受け取る。
<div style="background-color: #f0f0f0; padding: 10px; border-radius: 5px; color: black;">

$q$ <br>
$op_1$ <br>
$op_2$ <br>
$\dots$ <br>
$op_Q$
</div>

$op_i$は以下のいずれかである。
* `insert prev` $pos$ $val$ : ダミーセルの後ろから$pos$番目の位置に値$val$のセルを挿入する。もともと$pos$番目の位置にあったセルは、新しいセルの後ろにくる。
* `insert next` $pos$ $val$ : ダミーセルの前から$pos$番目の位置に値$val$のセルを挿入する。もともと$pos$番目の位置にあったセルは、新しいセルの前にくる。
* `delete prev` $pos$ : ダミーセルの後ろから$pos$番目の位置にあるセルを削除する。
* `delete next` $pos$ : ダミーセルの前から$pos$番目の位置にあるセルを削除する
* `display` : リストの内容を先頭から順に表示する
* `display-reverse` : リストの内容を末尾から順に表示する
* `search` $val$ : リストの中から値$val$を持つセルを探す。見つかった場合は`QUERY: $val found`というメッセージを末尾に改行をつけて出力する。ここで、`$val`はクエリに用いた値である。見つからなかった場合は`QUERY: $val not found`というメッセージを末尾に改行をつけて出力する。

#### 制約
* $1\leq Q \leq 20$
* $1\leq val \leq 10$

#### 具体例
##### 標準入力1
```text
20
insert next 1 1
display
insert prev 1 5
display
insert next 2 2
display
insert prev 2 4
display
insert next 3 3
display
delete next 1
display
delete prev 1
display
delete next 2
display
delete prev 2
display
delete next 1
display

```

##### 標準出力1
```text
1 
1 5 
1 2 5 
1 2 4 5 
1 2 3 4 5 
2 3 4 5 
2 3 4 
2 4 
4 

```

##### 標準入力2
```text
4
insert next 1 1
insert prev 1 2
search 1
search 3
```

##### 標準出力2
```text
QUERY: 1 found
QUERY: 3 not found
```

## I/O関連のユーティリティ関数の実装チェック
#### ソースコード test_util.c
```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "doublylinked_list.h"

int main(int argc, char *argv[]) {
  if (argc < 2) {
    fprintf(stderr, "Specify the option: io_array or io_file\n");
    exit(EXIT_FAILURE);
  }
  if (strcmp(argv[1], "io_array") == 0) {
    // read from array in stdin
    int n;
    scanf("%d", &n);
    int *data = (int *)malloc(n * sizeof(int));
    int i;
    for (i = 0; i < n; ++i) {
      scanf("%d", &data[i]);
    }
    Cell *head = CreateCell(0, true);
    ReadFromArray(head, data, n);
    Display(head);
    int *data_out = (int *)malloc(n * sizeof(int));
    WriteToArray(head->next, data_out, n);
    for (i = 0; i < n; ++i) {
      printf("%d ", data_out[i]);
    }
    printf("\n");
  } else if (strcmp(argv[1], "io_file") == 0) {
    if (argc < 3) {
      fprintf(stderr, "Specify the filename\n");
      exit(EXIT_FAILURE);
    }
    // read from file
    const char *filename = argv[2];
    Cell *head = CreateCell(0, true);
    ReadFromFile(head, filename);
    Display(head);
    WriteToFile(head->next, "output.txt");
    Cell *head2 = CreateCell(0, true);
    ReadFromFile(head2, "output.txt");
    Display(head2);
  }
  return 0;
}
```

このソースコードでは、第二引数に`io_array`か`io_file`を指定する。
* `io_array`を指定した場合は、標準入力から配列を読み込み、その内容を双方向循環リストに追加して`Display`関数で表示する。その後、リストの内容を配列に書き出し、その内容を標準出力に表示する。
  * 標準入力のフォーマットは、最初に配列の長さ$n$、続いて$n$個の整数が空白区切りで与えられる。
* `io_file`を指定した場合は、第三引数でファイル名を指定する。指定されたファイルから内容を読み込み、双方向循環リストに追加して`Display`関数で表示する。その後、リストの内容を`WriteToFile()`関数でファイルに出力し、その内容を再度`ReadFromFile()`関数で読み込み直して`Display`関数で表示する。

正常に動作するならば、標準出力には同じ内容の行が2回表示される。

#### 制約
* 入力される配列の長さ$n$は$1\leq n \leq 100$
* リストに含まれる要素の値は$1\leq x \leq 100$

#### 具体例1
##### 標準入力 stdin.txt
```text
5
1 2 3 4 5
```

##### 実行コマンド
```bash
$ ./test_util io_array < stdin.txt
1 2 3 4 5
1 2 3 4 5
```

#### 具体例2
##### 入力ファイル input.txt
```text
1
2
3
4
5
```

##### 実行コマンド
```bash
$ ./test_util io_file input.txt
1 2 3 4 5
1 2 3 4 5
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
