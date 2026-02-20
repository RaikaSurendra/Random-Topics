// Test multiple function calls
int square(int x) {
    return x * x;
}

int add(int a, int b) {
    return a + b;
}

int main() {
    int a = square(3);
    int b = square(4);
    return add(a, b);
}
