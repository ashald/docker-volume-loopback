#!/usr/bin/env bash

BASE_SIZE="100M"
BASE_FS="ext3"

testFallbackToDdFromFallocateOnUnsupportedFs() {
    local volume result count
    # setup

    ## for sparse=false we use fallocate and fallback to dd if fallocate is not supported
    volume=$(docker volume create -d "${DRIVER}" -o fs=xfs -o sparse=false -o size=50MiB 2>&1)
    result=$?

    ## because we shadow real data dir with our test volume we're sure there shouldn't be any volumes
    count=$(run ls -1 "${DATA_DIR}/" | grep -v "lost+found" | wc -l)

    assertEquals "Volume creation should succeed" "0" "${result}"
    assertEquals "There should be 1 volume" "1" "${count}"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testDdFailureScenarioWhenThereIsNotEnoughDiskSpace() {
    local error result count
    # setup

    ## attempt creating 2 GiB volume while we have only 1 GiB of space
    error=$(docker volume create -d "${DRIVER}" -o fs=xfs -o sparse=false -o size=200MiB 2>&1)
    result=$?

    ## because we shadow real data dir with our test volume we're sure there shouldn't be any volumes
    count=$(run ls -1 "${DATA_DIR}/" | grep -v "lost+found" | wc -l)

    assertEquals "Volume creation should fail" "1" "${result}"
    assertEquals "There should be no volumes" "0" "${count}"
}

. test.sh
