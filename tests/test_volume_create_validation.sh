#!/usr/bin/env bash

testWrongOptionNames() {
    local volume error result
    # setup
    error=$(docker volume create -d "${DRIVER}" -o x= -o y= 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail if unsupported options passed" "1" "${result}"
    assertContains "Error mentions wrong and correct options" "${error}" "options 'x, y' are not among supported ones: size, sparse, fs, uid, gid, mode"
}

testBelowMinAllowedSize() {
    local volume error result
    # setup
    error=$(docker volume create -d "${DRIVER}" -o size=19MB 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail for <=20MB" "1" "${result}"
    assertContains "Error mentions requested and min allowed size of 20MB" "${error}" "requested size '19000000' is smaller than minimum allowed 20MB"
}

testMinAllowedSize() {
    local volume result
    # setup
    volume=$(docker volume create -d "${DRIVER}" -o size=20MB)
    result=$?

    # checks
    assertEquals "Volume creation should succeed for 20MB" "0" "${result}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testWrongFs() {
    local volume error result
    # setup
    error=$(docker volume create -d "${DRIVER}" -o fs=foo 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail for unsupported FS" "1" "${result}"
    assertContains "Error mentions passed and supported FS options" "${error}" "only xfs and ext4 filesystems are supported, 'foo' requested"
}

testWrongName() {
    local volume error result
    # setup
    error=$(docker volume create -d "${DRIVER}" foo.bar 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail for invalid name" "1" "${result}"
    assertContains "Error mentions invalid name and allowed pattern" "${error}" "volume name 'foo.bar' does not match allowed pattern '^[a-zA-Z0-9][\w\-]{1,250}$'"
}

. test.sh
