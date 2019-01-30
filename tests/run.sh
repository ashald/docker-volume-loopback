#!/usr/bin/env bash

result=0

for suite in $(dirname ${0})/test_*; do
    echo "--- Executing '$(basename ${suite} .sh)' ---"

    ( cd $(dirname "${suite}"); ./"$(basename ${suite})" 2>&1; )

    test "${result}" -eq 0 -a $? -eq 0
    result=$?

    echo
done

echo -n "$(tput bold) ---> "

if [[ ${result} -eq 0 ]]; then
    echo "$(tput setaf 2)ALL TESTS PASSED"
else
    echo "$(tput setaf 1)SOME TESTS FAILED"
fi

exit ${result}
