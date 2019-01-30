#!/usr/bin/env bash

# A shortcut to export common things and trigger test execution

IMAGE="alpine"
DRIVER="docker-volume-loopback"

#random () {
#    echo "$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 32 | head -n 1)"
#}

#VOLUME="$(random)"

. ./shunit2