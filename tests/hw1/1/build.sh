#!/bin/bash

clang++ tests.cpp -c
clang++ *.o -o tests
./tests -s --order lex -r junit
