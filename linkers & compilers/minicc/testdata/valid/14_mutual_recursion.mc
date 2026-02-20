// Test mutual recursion — isEven/isOdd
// Two-pass semantic analysis registers all top-level functions first,
// so mutual recursion works without forward declarations.

int isEven(int n) {
    if (n == 0) {
        return 1;
    }
    return isOdd(n - 1);
}

int isOdd(int n) {
    if (n == 0) {
        return 0;
    }
    return isEven(n - 1);
}

int main() {
    return isEven(4);
}
