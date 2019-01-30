#!/usr/bin/env bash

oneTimeSetUp() {
    export VOLUME_DEFAULT=$(docker volume create -d "${DRIVER}")
    export VOLUME_CUSTOM=$(docker volume create -d "${DRIVER}" -o size=100Mi)
}


oneTimeTearDown() {
    docker volume rm "${VOLUME_DEFAULT}" "${VOLUME_CUSTOM}" > /dev/null
}

testDefaultVolumeSize() {
    size=$(docker run --rm -it -v "${VOLUME_DEFAULT}:/srv" "${IMAGE}" df -m /srv | tail -1 | awk '{print $2}')
    assertTrue "Volume size is less than 1024 MiB by default" "[ ${size} -lt 1024 ]"
}

testCustomVolumeSize() {
    size=$(docker run --rm -it -v "${VOLUME_CUSTOM}:/srv" "${IMAGE}" df -m /srv | tail -1 | awk '{print $2}')
    assertTrue "Volume size is less than requested value" "[ ${size} -lt 100 ]"
}

. test.sh
