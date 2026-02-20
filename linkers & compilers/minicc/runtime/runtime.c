// runtime.c — Mini-C runtime support
// Linked with every compiled Mini-C program.
//
// Provides:
//   - print_int(int x): prints an integer to stdout
//   - main(): calls mc_main (the compiled Mini-C main) and returns its result
//
// On macOS, symbol names are prefixed with underscore (_mc_main, _print_int).
// On Linux, no prefix (mc_main, print_int).

#include <stdio.h>

void print_int(int x) {
    printf("%d\n", x);
}

// The Mini-C compiler emits the user's main() as mc_main / _mc_main
// to avoid collision with the C runtime's main().
extern int mc_main(void);

int main(void) {
    int result = mc_main();
    return result;
}
