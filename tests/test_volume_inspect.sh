#!/usr/bin/env bash


testRegularXfsStatus() {
    local volume info status_fs status_size_max status_size_allocated
    # setup

    volume=$(docker volume create -d "${DRIVER}" -o fs=xfs -o sparse=false -o size=100MB)

    info=$(docker volume inspect "${volume}" | jq ".[0].Status")

    status_fs=$(echo "${info}" | jq -r ".fs")
    status_size_max=$(echo "${info}" | jq -r '.["size-max"]')
    status_size_allocated=$(echo "${info}" | jq -r '.["size-allocated"]')

    assertEquals "Reported FS check" "xfs" "${status_fs}"
    assertEquals "Reported max size check" "100000000" "${status_size_max}"
    assertTrue "Reported allocated size check" "[ ${status_size_allocated} -ge 100000000 ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testSparseExt4Status() {
    local volume info status_fs status_size_max status_size_allocated
    # setup

    volume=$(docker volume create -d "${DRIVER}" -o fs=ext4 -o sparse=true -o size=100MB)

    info=$(docker volume inspect "${volume}" | jq ".[0].Status")

    status_fs=$(echo "${info}" | jq -r ".fs")
    status_size_max=$(echo "${info}" | jq -r '.["size-max"]')
    status_size_allocated=$(echo "${info}" | jq -r '.["size-allocated"]')

    assertEquals "Reported FS check" "ext4" "${status_fs}"
    assertEquals "Reported max size check" "100000000" "${status_size_max}"
    assertTrue "Reported allocated size check" "[ ${status_size_allocated} -lt 100000000 ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

. test.sh
