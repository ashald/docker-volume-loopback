#!/usr/bin/env bash

testDefaultMode() {
    local volume mode
    # setup
    volume=$(docker volume create -d "${DRIVER}")

    # stat produces some funny characters at the end so we want to clean them up
    mode=$(docker run --rm -it -v "${volume}:/vol" "${IMAGE}" stat -c '%a ' /vol | tr -cd '[0-7]')

    # check
    assertEquals "Default mode should be 755" "755" "${mode}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}


testCustomMode() {
    local volume mode
    # setup
    volume=$(docker volume create -d "${DRIVER}" -o mode=7777)

    # stat produces some funny characters at the end so we want to clean them up
    mode=$(docker run --rm -it -v "${volume}:/vol" "${IMAGE}" stat -c '%a ' /vol | tr -cd '[0-7]')

    # check
    assertEquals "Mode should be 7777" "7777" "${mode}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testZeroMode() {
    local volume info error result
    # setup
    error=$(docker volume create -d "${DRIVER}" -o mode=0 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail" "1" "${result}"
    assertContains "${error}" "mode value '0' does not fall between 0 and 7777 in octal encoding"
}

testInvalidMode() {
    local volume info error result
    # setup
    error=$(docker volume create -d "${DRIVER}" -o mode=x 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail" "1" "${result}"
    assertContains "${error}" "cannot parse mode 'x' as positive 4-position octal: strconv.ParseUint: parsing \"x\": invalid syntax"
}


. test.sh
