#include <cstdlib>

typedef struct User {
	int id;
	int age;
	int number;
} User;

int main() {
    void *user1 = new User{
        id: 1,
        age: 22,
        number: 333,
    };

    void *user2 = std::calloc(0, sizeof(User));
    std::memcpy(user2, user1, sizeof(User));
}