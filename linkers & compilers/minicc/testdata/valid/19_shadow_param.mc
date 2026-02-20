// Test parameter shadowed by local variable
int foo(int x) {
    int x = 7;
    return x;
}

int main() {
    return foo(99);
}
