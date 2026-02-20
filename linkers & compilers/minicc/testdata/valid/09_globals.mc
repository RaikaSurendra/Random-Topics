// Test global variable access
int g = 1;

int add_to_global(int n) {
    g = g + n;
    return g;
}

int main() {
    add_to_global(2);
    add_to_global(3);
    return g;
}
