// Test chained function calls f(g(h(x)))
int add1(int x) {
    return x + 1;
}

int double(int x) {
    return x * 2;
}

int square(int x) {
    return x * x;
}

int main() {
    return add1(square(double(1)));
}
