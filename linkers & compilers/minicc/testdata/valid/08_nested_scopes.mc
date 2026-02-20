// Test block scoping and shadowing
int main() {
    int x = 1;
    {
        int x = 2;
        {
            int x = 3;
            x = x + 10;
        }
        x = x + 100;
    }
    return x;
}
