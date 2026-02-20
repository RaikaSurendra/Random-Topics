// Test boolean operators
int main() {
    int a = 5;
    int b = 10;
    int c = (a < b) && (b > 0);
    int d = (a > b) || (a == 5);
    int e = !(a == b);
    return c + d + e;
}
