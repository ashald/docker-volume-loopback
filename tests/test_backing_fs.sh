#!/usr/bin/env bash

BASE_SIZE="100M"
BASE_FS="ext3"

testFallbackToDdFromFallocateOnUnsupportedFs() {
    local error result count
    # setup

    ## for sparse=false we use fallocate and fallback to dd if fallocate is not supported
    error=$(docker volume create -d "${DRIVER}" -o fs=xfs -o sparse=false -o size=50MiB 2>&1)
    result=$?

    ## because we shadow real data dir with our test volume we're sure there shouldn't be any volumes
    count=$(ls -1 "/var/lib/${DRIVER}/" | grep -v "lost+found" | wc -l)

    assertEquals "0" "${result}"
    assertEquals "1" "${count}"
}

testDdFailureScenarioWhenThereIsNotEnoughDiskSpace() {
    local error result count
    # setup

    ## attempt creating 2 GiB volume while we have only 1 GiB of space
    error=$(docker volume create -d "${DRIVER}" -o fs=xfs -o sparse=false -o size=200MiB 2>&1)
    result=$?

    ## because we shadow real data dir with our test volume we're sure there shouldn't be any volumes
    count=$(ls -1 "/var/lib/${DRIVER}/" | grep -v "lost+found" | wc -l)

    assertEquals "1" "${result}"
    assertEquals "0" "${count}"
}

. test.sh
