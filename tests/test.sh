#!/usr/bin/env bash

# A shortcut to export common things and trigger test execution

IMAGE="alpine"
DRIVER="docker-volume-loopback"
DATA_DIR="/var/lib/${DRIVER}"

oneTimeSetUp() {
    docker volume rm $(docker volume create -d "${DRIVER}" -o size=100MiB) &> /dev/null
}

setUp() {
    HANDLE=$(mktemp -u)
    truncate -s "${BASE_SIZE:-2G}" "${HANDLE}"


    case "${BASE_FS:-xfs}" in
    xfs)
        mkfs.xfs "${HANDLE}" &> /dev/null
        mount -o nouuid "${HANDLE}" "${DATA_DIR}"
        ;;
    ext*)
        mkfs.${BASE_FS} -F "${HANDLE}" &> /dev/null
        mount "${HANDLE}" "${DATA_DIR}"
        ;;
    *)
        echo "Unsupported BASE fs"
        exit 1
        ;;
    esac
}

tearDown() {
    umount -ld "${DATA_DIR}"
    rm -f "${HANDLE}"
}

. ./shunit2
