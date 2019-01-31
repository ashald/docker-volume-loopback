#!/usr/bin/env bash

testDefaultVolumeSize() {
    local volume size
    # setup
    volume=$(docker volume create -d "${DRIVER}")
    local size=$(docker run --rm -it -v "${volume}:/srv" "${IMAGE}" df -m /srv | tail -1 | awk '{print $2}')

    # checks
    assertTrue "Volume size is less than 1024 MiB by default" "[ ${size} -lt 1024 ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

testCustomVolumeSize() {
    local volume size
    # setup
    volume=$(docker volume create -d "${DRIVER}" -o size=100Mi)
    size=$(docker run --rm -it -v "${volume}:/srv" "${IMAGE}" df -m /srv | tail -1 | awk '{print $2}')

    # checks
    assertTrue "Volume size is less than requested value" "[ ${size} -lt 100 ]"

    # cleanup
    docker volume rm "${volume}" > /dev/null
}

. test.sh
