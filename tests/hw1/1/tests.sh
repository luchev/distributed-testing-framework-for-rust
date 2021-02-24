#!/bin/bash

clang++ *.o -o tests

./tests -s --order lex -r junit > junit_results

xunit-viewer -r junit_results -o junit_results.html
