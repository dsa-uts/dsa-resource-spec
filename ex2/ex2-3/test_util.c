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