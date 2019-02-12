#!/usr/bin/env bash

FS="xfs"

testRegularVolumeChecksDiskSpaceBeforeFormatting() {
    local error result count
    # setup

    ## attempt creating 10 GiB volume while we have only 2 GiB of space
    error=$(docker volume create -d "${DRIVER}" -o fs=${FS} -o sparse=false -o size=10GiB 2>&1)
    result=$?

    ## because we shadow real data dir with our test volume we're sure there shouldn't be any volumes
    count=$(run ls -1 "${DATA_DIR}/" | wc -l)

    # checks
    assertEquals "1" "${result}"
    assertEquals "0" "${count}"
}

testRegularVolumeReservesDiskSpace() {
    local volume info allocated_size apparent_size
    # setup
    local volume=$(docker volume create -d "${DRIVER}" -o fs=${FS} -o sparse=false)

    allocated_size="$(($(run stat -c '%b %B' ${DATA_DIR}/${volume} | tr ' ' '*')))" # allocated space in bytes
    apparent_size="$(run stat -c '%s' ${DATA_DIR}/${volume})"                       # apparent  space in bytes

    # checks
    assertTrue "Regular ${FS} volume of ${apparent_size} MiB should take at least same space: ${allocated_size} MiB" "[ ${allocated_size} -ge ${apparent_size} ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testSparseVolumeDoesNotCheckAvailableDiskSpace() {
    local volume result info apparent_size
    # setup

    ## attempt creating 10 GiB volume while we have only 2 GiB of space
    volume=$(docker volume create -d "${DRIVER}" -o fs=${FS} -o sparse=true -o size=10GiB)
    result=$?

    apparent_size="$(run stat -c '%s' ${DATA_DIR}/${volume})"                       # apparent  space in bytes

    # checks
    assertEquals "0" "${result}"
    assertEquals "$((10*1024*1024*1024))" "${apparent_size}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testSparseVolumeDoesNotTakeDiskSpace() {
    local volume info allocated_size apparent_size
    # setup
    local volume=$(docker volume create -d "${DRIVER}" -o fs=${FS} -o sparse=true)

    allocated_size="$(($(run stat -c '%b %B' ${DATA_DIR}/${volume} | tr ' ' '*')))" # allocated space in bytes
    apparent_size="$(run stat -c '%s' ${DATA_DIR}/${volume})"                       # apparent  space in bytes

    # checks
    assertTrue "Sparse ${FS} volume of ${apparent_size} MiB should take less space: ${allocated_size} MiB" "[ ${allocated_size} -lt ${apparent_size} ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

. test.sh
