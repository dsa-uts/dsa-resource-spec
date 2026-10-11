# 概要
[課題リンク](https://www.coins.tsukuba.ac.jp/~amagasa/lecture/dsa-jikken/report1/#_5)

`gcd_euclid` を，再帰的アルゴリズム（教科書p. 14）に基づいて書き換えた `gcd_recursive.c` および `main_recursive.c` を作成しなさい．関数名は `gcd_recursive` とする．

# ファイル
## gcd_recursive.c
```c
#include <stdio.h>
#include <stdlib.h>

// Find the greatest common divisor of the two integers, n and m.
int gcd_recursive(int n, int m) {

    // 関数を完成させよ

    return n;
}
```

## main_recursive.c
```c
#include <stdio.h>
#include <stdlib.h>

// Find the greatest common divisor of the two integers, n and m.
int gcd_recursive(int, int);

int main(int argc, char *argv[]) {
  // Process arguments.
  if (argc != 3) {
    fprintf(stderr, "Usage: %s <number1> <number2>\n", argv[0]);
    return EXIT_FAILURE;
  }
  int n = atoi(argv[1]);
  int m = atoi(argv[2]);

  // Compute and output the greatest common divisor.
  int gcd = gcd_recursive(n, m);
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
