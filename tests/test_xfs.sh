#!/usr/bin/env bash

FS="xfs"

oneTimeSetUp() {
    export VOLUME_SPARSE=$(docker volume create -d "${DRIVER}" -o fs=${FS} -o sparse=true)
    export VOLUME_REGULAR=$(docker volume create -d "${DRIVER}" -o fs=${FS} -o sparse=false)
}


oneTimeTearDown() {
    docker volume rm "${VOLUME_SPARSE}" > /dev/null
    docker volume rm "${VOLUME_REGULAR}" > /dev/null
}

testSparseVolumeDoesNotTakeDiskSpace() {
    info=$(ls --block-size=M -ls "/var/lib/${DRIVER}/${VOLUME_SPARSE}.${FS}")
    allocated_size=$(echo ${info} | awk '{print $1}' | tr -dc '0-9')
    apparent_size=$(echo ${info} | awk '{print $6}' | tr -dc '0-9')

    assertTrue "Sparse ${FS} volume of ${apparent_size} MiB should take less space: ${allocated_size} MiB" "[ ${allocated_size} -lt ${apparent_size} ]"
}

testRegularVolumeReservesDiskSpace() {
    info=$(ls --block-size=M -ls "/var/lib/${DRIVER}/${VOLUME_REGULAR}.${FS}")
    allocated_size=$(echo ${info} | awk '{print $1}' | tr -dc '0-9')
    apparent_size=$(echo ${info} | awk '{print $6}' | tr -dc '0-9')

    assertTrue "Regular ${FS} volume of ${apparent_size} MiB should take at least same space: ${allocated_size} MiB" "[ ${allocated_size} -ge ${apparent_size} ]"
}

. test.sh
