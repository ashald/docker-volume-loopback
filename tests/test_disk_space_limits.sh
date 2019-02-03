#!/usr/bin/env bash

testDefaultVolumeSize() {
    local volume size
    # setup
    volume=$(docker volume create -d "${DRIVER}")

    ## Filesystem           1M-blocks      Used Available Use% Mounted on
    ## /dev/loop0                1014        32       982   3% /srv
    size=$(docker run --rm -it -v "${volume}:/srv" "${IMAGE}" df -m /srv | tail -1 | awk '{print $2}')

    # checks
    assertTrue "Mounted volume size should be <=1024 MiB" "[ ${size} -le 1024 ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testCustomVolumeSize() {
    local volume size
    # setup
    volume=$(docker volume create -d "${DRIVER}" -o size=100MiB)

    ## Filesystem           1M-blocks      Used Available Use% Mounted on
    ## /dev/loop0                  91         5        86   5% /srv
    size=$(docker run --rm -it -v "${volume}:/srv" "${IMAGE}" df -m /srv | tail -1 | awk '{print $2}')

    # checks
    assertTrue "Mounted volume size should be <=100MiB" "[ ${size} -le 100 ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

. test.sh
