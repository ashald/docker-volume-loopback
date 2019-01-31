#!/usr/bin/env bash

# A shortcut to export common things and trigger test execution

IMAGE="alpine"
DRIVER="docker-volume-loopback"
DATA_DIR="/var/lib/${DRIVER}"


setUp() {
    HANDLE=$(mktemp -u)
    truncate -s "${1:-2GiB}" "${HANDLE}"
    mkfs.xfs "${HANDLE}" &> /dev/null
    mount -o nouuid "${HANDLE}" "${DATA_DIR}"
}

tearDown() {
    umount -ld "${DATA_DIR}"
    rm -f "${HANDLE}"
}

. ./shunit2
