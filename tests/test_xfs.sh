#!/usr/bin/env bash

FS="xfs"

testSparseVolumeDoesNotTakeDiskSpace() {
    local volume info allocated_size apparent_size
    # setup
    local volume=$(docker volume create -d "${DRIVER}" -o fs=${FS} -o sparse=true)
    local info=$(ls --block-size=M -ls "/var/lib/${DRIVER}/${volume}.${FS}")
    local allocated_size=$(echo ${info} | awk '{print $1}' | tr -dc '0-9')
    local apparent_size=$(echo ${info} | awk '{print $6}' | tr -dc '0-9')

    # checks
    assertTrue "Sparse ${FS} volume of ${apparent_size} MiB should take less space: ${allocated_size} MiB" "[ ${allocated_size} -lt ${apparent_size} ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testRegularVolumeReservesDiskSpace() {
    local volume info allocated_size apparent_size
    # setup
    local volume=$(docker volume create -d "${DRIVER}" -o fs=${FS} -o sparse=false)
    local info=$(ls --block-size=M -ls "/var/lib/${DRIVER}/${volume}.${FS}")
    local allocated_size=$(echo ${info} | awk '{print $1}' | tr -dc '0-9')
    local apparent_size=$(echo ${info} | awk '{print $6}' | tr -dc '0-9')

    # checks
    assertTrue "Regular ${FS} volume of ${apparent_size} MiB should take at least same space: ${allocated_size} MiB" "[ ${allocated_size} -ge ${apparent_size} ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

. test.sh
