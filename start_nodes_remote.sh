#!/bin/bash

difficulty=1
capacity=10
hostnames=( "dclass0" "dclass1" "dclass2" "dclass3" "dclass4" "dclass0" "dclass1" "dclass2" "dclass3" "dclass4")

HOSTPORT=7070
APIPORT=9090

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [N(default=1)]]"
    exit 0
fi

# First argument is the number of nodes to run, default 1
N=${1:-0}
if [ $N -ge 1 ]; then
    N=$((N-1))
fi

echo "Sending binaries..."
for host in "${hostnames[@]:1:5}"
do 
    scp ~/noobcash-cli $host:~/noobcash-cli
    ssh $host chmod +x ~/noobcash-cli
    scp ~/noobcash-node $host:~/noobcash-node
    ssh $host chmod +x ~/noobcash-node
done

CMD=""
for i in $(seq 0 $N); do
    myHost="${hostnames[$i]}"
    myPort=$((HOSTPORT + i))
    myHostString="$myHost:$myPort"
    myApiPort=$((APIPORT + i))

    sedIdentifier="'s/^/[ApiPort: $myApiPort] /'"
    # echo $sedIdentifier

    myCmd="ssh $myHost ~/noobcash-node"
    if [ $i -eq 0 ]; then
        myCmd="$myCmd bootstrap --nodecnt $((N+1))"
        myBootstrap="$myHostString"
    fi
    myCmd="$myCmd --hostname $myHostString --apiport $myApiPort --difficulty $difficulty --capacity $capacity --bootstrap $myBootstrap"
    # echo $myCmd
    if [ $i -ne 0 ]; then
        CMD="$CMD & sleep 2;"
    fi
    CMD="$CMD $myCmd 2>&1 >/dev/null | sed -e $sedIdentifier"
done
#echo $CMD

function kill_nodes() {
    for host in "${hostnames[@]:0:5}"
    do 
        eval "ssh $host \"pgrep -f noobcash-node | xargs kill\""
    done
}

trap "kill_nodes" SIGINT
eval $CMD
