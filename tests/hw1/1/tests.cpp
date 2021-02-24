#include <cstring>
#include <iostream>

#include "MyString.h"
#include "catch.hpp"

TEST_CASE("Setter/Getter", "[set]") {
    MyString s("asd");
    REQUIRE(123 == 123);
}

TEST_CASE("Constructor", "[set]") {
    MyString s("asd");
    REQUIRE(1 == 1);
}

TEST_CASE("Test 2", "[set]") {
    MyString s("asd");
    s = "abc";
    REQUIRE(strcmp(s.get(), "abc") == 0);
    s = "12312312";
    REQUIRE(strcmp(s.get(), "12312312") == 0);
}

TEST_CASE("Append", "[get]") {
    MyString s("asd");
    s.append('x');
    REQUIRE(strcmp(s.get(), "abcx") == 0);
}
