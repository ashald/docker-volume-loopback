#!/usr/bin/env bash


testListVolumes() {
    local volume info
    # setup

    volume1=$(docker volume create -d "${DRIVER}" -o fs=ext4 -o sparse=true -o size=100MB vol01)
    volume2=$(docker volume create -d "${DRIVER}" -o fs=ext4 -o sparse=true -o size=100MB vol02)

    info=$(docker volume ls | grep "${DRIVER}" | awk '{print $2}' | xargs echo)

    assertEquals "Check volume list" "${volume1} ${volume2}" "${info}"

    # cleanup
    docker volume rm "${volume1}" > /dev/null
    docker volume rm "${volume2}" > /dev/null
}

. test.sh
