#!/bin/bash

PANOS_HOSTNAME=$1
PANOS_USERNAME=$2
PANOS_PASSWORD=$3
SSH_PRIVATE_KEY=$4

export PANOS_HOSTNAME PANOS_USERNAME PANOS_PASSWORD

while true; do
    ignore=`go run fwinit.go ${SSH_PRIVATE_KEY} 2>&1`
    if [ $? -eq 0 ]; then
        echo "Firewall initial config is done"
        break
    fi

    sleep 10
done
