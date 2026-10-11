# 概要
[課題リンク](https://www.coins.tsukuba.ac.jp/~amagasa/lecture/dsa-jikken/report1/#_4)

教科書リスト1-4（p. 7）の「ユークリッドの互除法」に基づいたプログラム`gcd_euclid.c`および`main_euclid.c`を作成しなさい。

# ファイル
## gcd_euclid.c
```c
#include <stdio.h>
#include <stdlib.h>

// Find the greatest common divisor of the two integers, n and m.
int gcd_euclid(int n, int m) {

    // 関数を完成させよ

    return n;
}
```

## main_euclid.c
```c
#include <stdio.h>
#include <stdlib.h>

// Find the greatest common divisor of the two integers, n and m.
int gcd_euclid(int, int);

int main(int argc, char *argv[]) {
  // Process arguments.
  if (argc != 3) {
    fprintf(stderr, "Usage: %s <number1> <number2>\n", argv[0]);
    return EXIT_FAILURE;
  }
  int n = atoi(argv[1]);
  int m = atoi(argv[2]);

  // Compute and output the greatest common divisor.
  int gcd = gcd_euclid(n, m);
  printf("The GCD of %d and %d is %d.\n", n, m, gcd);

  return EXIT_SUCCESS;
}
```

## Makefile
```make
all: gcd_iter gcd_euclid gcd_recursive

gcd_iter: gcd_iter.o main_iter.o

gcd_euclid: gcd_euclid.o main_euclid.o

gcd_recursive: gcd_recursive.o main_recursive.o
```
