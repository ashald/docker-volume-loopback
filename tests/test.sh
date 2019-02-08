#!/usr/bin/env bash

# A shortcut to export common things and trigger test execution

IMAGE="alpine"
DRIVER="docker-volume-loopback"
eval $(strings -a /proc/$(pidof docker-volume-loopback)/environ | grep DATA_DIR)
DATA_DIR=${DATA_DIR:-"/var/lib/${DRIVER}"} # a default fall-back

run() {
    nsenter -t $(pidof "${DRIVER}") -a "${@}"
}

oneTimeSetUp() {
    docker volume rm $(docker volume create -d "${DRIVER}" -o size=100MiB) &> /dev/null
}

setUp() {
    HANDLE=$(mktemp -u)
    run truncate -s "${BASE_SIZE:-2G}" "${HANDLE}"


    case "${BASE_FS:-xfs}" in
    xfs)
        run mkfs.xfs -f "${HANDLE}" &> /dev/null
        run mount -o nouuid "${HANDLE}" "${DATA_DIR}"
        ;;
    ext*)
        run mkfs.${BASE_FS} -F "${HANDLE}" &> /dev/null
        run mount "${HANDLE}" "${DATA_DIR}"
        ;;
    *)
        echo "Unsupported BASE fs"
        exit 1
        ;;
    esac
}

tearDown() {
    run umount -ld "${DATA_DIR}"
    run rm -f "${HANDLE}"
}

. ./shunit2
