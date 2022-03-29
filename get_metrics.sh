#!/bin/bash

hostnames=( "dclass0" "dclass1" "dclass2" "dclass3" "dclass4" "dclass0" "dclass1" "dclass2" "dclass3" "dclass4")

HOSTPORT=7070
APIPORT=9090

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [N(default=5)]]"
    exit 0
fi

# First argument is the number of nodes to run, default 1
N=${1:-5}
if [ $N != 5 ] && [ $N == 10 ]; then
    echo "$N is not a correct value. Specify 5 or 10."
    exit 0
fi
N=$((N-1))

for i in $(seq 0 $N); do
    myHost="${hostnames[$i]}"
    myApiPort=$((APIPORT + i))
    myHostString="$myHost:$myApiPort"

    eval "./noobcash-cli balance -a $myHostString"
done

for i in $(seq 0 $N); do
    myHost="${hostnames[$i]}"
    myApiPort=$((APIPORT + i))
    myHostString="$myHost:$myApiPort"

    eval "./noobcash-cli stats -a $myHostString"
done

for i in $(seq 0 $N); do
    myHost="${hostnames[$i]}"
    myPort=$((HOSTPORT + i))
    myHostString="$myHost:$myPort"

    eval "curl -X POST http://$myHostString/chain-length && echo "
done
