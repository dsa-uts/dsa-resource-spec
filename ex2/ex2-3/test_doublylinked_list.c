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