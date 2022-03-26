#!/bin/bash

hostnames=( "dclass0" "dclass1" "dclass2" "dclass3" "dclass4" "dclass0" "dclass1" "dclass2" "dclass3" "dclass4")

APIPORT=9090

if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [N(default=5, {5,10})] [-w waits (default=no wait)]" 
    exit 0
fi

# First argument is the number of nodes to run, default 5
N=${1:-5}
if [ $N == 5 ]; then
    TXPATH="~/5nodes"
elif [ $N == 10 ]; then
    TXPATH_="~/10nodes"
else 
    echo "$N is not a correct value. Specify 5 or 10."
    exit 0
fi
N=$((N-1))

WAIT=""
if [ "$2" = "--wait" ] || [ "$2" = "-w" ]; then
    WAIT="-w"
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
    myApiPort=$((APIPORT + i))
    myHostString="$myHost:$myApiPort"
    myPath="$TXPATH/transactions$i.txt"

    myCmd="ssh $myHost ~/noobcash-cli s $myPath --address $myHostString $WAIT"
    # echo $myCmd
    if [ $i -ne 0 ]; then
        CMD="$CMD & sleep 2;"
    fi
    CMD="$CMD $myCmd 2>&1"
done
echo $CMD

function kill_nodes() {
    for host in "${hostnames[@]:0:5}"
    do 
        eval "ssh $host \"pgrep -f noobcash-cli | xargs kill\""
    done
}

trap "kill_nodes" SIGINT
eval $CMD
