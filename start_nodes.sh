#!/bin/bash

difficulty=1
capacity=10

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [N(default=1)]]"
    exit 0
fi

# First argument is the number of nodes to run, default 1
N=${1:-0}
if [ $N -ge 1 ]; then
    N=$((N-1))
fi

HOSTNAME=localhost
HOSTPORT=7070
APIPORT=9090

echo "Building..."
make

CMD=""
for i in $(seq 0 $N); do
    myPort=$((HOSTPORT + i))
    myHostString="$HOSTNAME:$myPort"
    myApiPort=$((APIPORT + i))

    sedIdentifier="'s/^/[ApiPort: $myApiPort] /'"
    # echo $sedIdentifier
    myCmd="./bin/noobcash-node"
    if [ $i -eq 0 ]; then
        myCmd="$myCmd bootstrap"
    fi
    myCmd="$myCmd --hostname $myHostString --apiport $myApiPort --difficulty $difficulty --capacity $capacity"
    # echo $myCmd
    if [ $i -ne 0 ]; then
        CMD="$CMD & sleep 2;"
    fi
    CMD="$CMD $myCmd 2>&1 >/dev/null | sed -e $sedIdentifier"
done
# ps | grep noobcash-node | grep -v 'grep' | cut -d' ' -f2 | xargs kill
trap "ps | grep noobcash-node | grep -v 'grep' | cut -d' ' -f1 | xargs kill" SIGINT
eval $CMD