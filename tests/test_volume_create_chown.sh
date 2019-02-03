#!/usr/bin/env bash

testDefaultOwner() {
    local volume uid gid
    # setup
    volume=$(docker volume create -d "${DRIVER}")

    info=$(docker run --rm -it -v "${volume}:/vol" "${IMAGE}" ls -lan | grep vol)
    uid=$(echo "${info}" | awk '{print $3}')
    gid=$(echo "${info}" | awk '{print $4}')

    # checks
    assertEquals "Default user owner is root (uid=0)" "0" "${uid}"
    assertEquals "Default group owner is root (gid=0)" "0" "${gid}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testEmptyValuesIgnored() {
    local volume uid gid
    # setup
    volume=$(docker volume create -d "${DRIVER}" -o uid= -o gid=)

    info=$(docker run --rm -it -v "${volume}:/vol" "${IMAGE}" ls -lan | grep vol)
    uid=$(echo "${info}" | awk '{print $3}')
    gid=$(echo "${info}" | awk '{print $4}')

    # checks
    assertEquals "User owner should be 0" "0" "${uid}"
    assertEquals "Group owner should be 0" "0" "${gid}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testChangeUser() {
    local volume info uid gid
    # setup
    volume=$(docker volume create -d "${DRIVER}" -o uid=123)
    info=$(docker run --rm -it -v "${volume}:/vol" "${IMAGE}" ls -lan | grep vol)
    uid=$(echo "${info}" | awk '{print $3}')
    gid=$(echo "${info}" | awk '{print $4}')

    # checks
    assertEquals "User owner should be changed to 123" "123" "${uid}"
    assertEquals "Group owner should remain 0" "0" "${gid}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testChangeGroup() {
    local volume info uid gid
    # setup
    volume=$(docker volume create -d "${DRIVER}" -o gid=456)
    info=$(docker run --rm -it -v "${volume}:/vol" "${IMAGE}" ls -lan | grep vol)
    uid=$(echo "${info}" | awk '{print $3}')
    gid=$(echo "${info}" | awk '{print $4}')

    # checks
    assertEquals "User owner should remain 0" "0" "${uid}"
    assertEquals "Group owner should be changed to 456" "456" "${gid}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testChangeUserAndGroup() {
    local volume info uid gid
    # setup
    volume=$(docker volume create -d "${DRIVER}" -o uid=123 -o gid=456)
    info=$(docker run --rm -it -v "${volume}:/vol" "${IMAGE}" ls -lan | grep vol)
    uid=$(echo "${info}" | awk '{print $3}')
    gid=$(echo "${info}" | awk '{print $4}')

    # checks
    assertEquals "User owner should be changed to 123" "123" "${uid}"
    assertEquals "Group owner should be changed to 456" "456" "${gid}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}


testNegativeUid() {
    local volume info error result
    # setup
    error=$(docker volume create -d "${DRIVER}" -o uid=-1 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail" "1" "${result}"
    assertContains "${error}" "'uid' option should be >= 0 but received '-1'"
}

testNonIntegerUid() {
    local volume info error result
    # setup
    error=$(docker volume create -d "${DRIVER}" -o uid=x 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail" "1" "${result}"
    assertContains "${error}" "cannot parse 'uid' option value 'x' as an integer: strconv.Atoi: parsing \"x\": invalid syntax"
}

testNegativeGid() {
    local volume info error result
    # setup
    error=$(docker volume create -d "${DRIVER}" -o gid=-1 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail" "1" "${result}"
    assertContains "${error}" "'gid' option should be >= 0 but received '-1'"
}

testNonIntegerGid() {
    local volume info error result
    # setup
    error=$(docker volume create -d "${DRIVER}" -o gid=x 2>&1)
    result=$?

    # checks
    assertEquals "Volume creation should fail" "1" "${result}"
    assertContains "${error}" "cannot parse 'gid' option value 'x' as an integer: strconv.Atoi: parsing \"x\": invalid syntax"
}


. test.sh
