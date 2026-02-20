// Test 5-level nested blocks
int main() {
    int x = 0;
    {
        int a = 10;
        {
            int b = 20;
            {
                int c = 30;
                {
                    int d = 40;
                    x = a + b + c + d;
                }
            }
        }
    }
    return x;
}
