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
