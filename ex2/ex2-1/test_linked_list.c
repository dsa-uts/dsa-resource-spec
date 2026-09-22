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
