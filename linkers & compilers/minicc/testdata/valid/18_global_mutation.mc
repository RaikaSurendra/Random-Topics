// Test global state mutation across function calls
int counter = 0;

int increment(int n) {
    counter = counter + n;
    return counter;
}

int main() {
    increment(5);
    increment(10);
    return counter;
}
