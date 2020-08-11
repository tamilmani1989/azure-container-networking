#!/bin/bash

# Read from Command Line
NET_COUNT=${1}
SUBNETS_PER_NET=${2}
CONTAINERS_PER_SUBNET=${3}
CONTAINERS_PER_NODE=${4}

# Ensure Limits Honored
if [[ "$2" -gt 3 ]]
then
echo "Maximum 3 Subnets Per Network"
SUBNETS_PER_NET=3
fi

if [[ "$3" -gt 1000000 ]]
then
echo "Maximum 1 million Containers Per Subnet"
CONTAINERS_PER_SUBNET=1000000
fi

if [[ "$4" -gt 3 ]]
then
echo "Maximum 3 Containers Per Node"
CONTAINERS_PER_NODE = 3
fi

SUBNET_COUNT=$(($NET_COUNT*$SUBNETS_PER_NET))
CONTAINER_COUNT=$(($NET_COUNT*$SUBNETS_PER_NET*$CONTAINERS_PER_SUBNET))
NODE_COUNT=$(($CONTAINER_COUNT/$CONTAINERS_PER_NODE+1))

echo "Creating Test Plan with ${NET_COUNT} Networks, ${SUBNETS_PER_NET} Subnets Per Network, ${CONTAINERS_PER_SUBNET} Containers Per Subnet, and ${CONTAINERS_PER_NODE} Containers Per Node."

# Initializes ids
NODE_IDS=()
NET_IDS=()
SUBNET_IDS=()
CONTAINER_IDS=()

# Initialize Node IDs
for (( c=1; c<=$NODE_COUNT; c++ ))
do
	NODE_IDS+=($(od -x /dev/urandom | head -1 | awk '{OFS="-"; print $2$3,$4,$5,$6,$7$8$9}'))
done

# Initialize Subnet IDs
for ((c=1; c<=$SUBNETS_PER_NET; c++ ))
do
	SUBNET_IDS+=("subnet${c}")
done

# Initialize Net IDs
for ((c=1; c<=$NET_COUNT; c++ ))
do
	NET_IDS+=($(od -x /dev/urandom | head -1 | awk '{OFS="-"; print $2$3,$4,$5,$6,$7$8$9}'))
done

# Initialize Container IDs
for ((c=1; c<=$CONTAINER_COUNT; c++ ))
do
	CONTAINER_IDS+=($(od -x /dev/urandom | head -1 | awk '{OFS="-"; print $2$3,$4,$5,$6,$7$8$9}'))
done

# Show Array Contents
echo "${NET_IDS[*]}"

# Clear out old CSV Files

rm nets.csv
rm subnets.csv
rm containers.csv
rm nodes.csv

# Create New CSV Files

# nodes.csv (node_id, ip_address, orchestrator_type, location_type)
for (( c=0; c<$NODE_COUNT; c++ ))
do
	echo "${NODE_IDS[c]},127.0.0.1,ServiceFabric,dummy_location" >> nodes.csv
done

# nets.csv (net_id, net_type)
for (( c=0; c<$NET_COUNT; c++ ))
do
	echo "${NET_IDS[c]},AzureNet" >> nets.csv
done

# subnets.csv (subnet_id, net_id)
for (( c=0; c<$NET_COUNT; c++ ))
do
	for (( d=0; d<$SUBNETS_PER_NET; d++ ))
	do
		echo "${SUBNET_IDS[d]},${NET_IDS[c]}" >> subnets.csv
	done
done

# containers.csv (subnet_id, net_id, container_id, node_id, pod_name, pod_namespace)
x=0
n=0
for (( i=0; i<$NET_COUNT; i++ ))
do
	for (( j=0; j<$SUBNETS_PER_NET; j++ ))
	do
		for (( k=0; k<$CONTAINERS_PER_SUBNET; k++ ))
		do
			echo "${SUBNET_IDS[j]},${NET_IDS[i]},${CONTAINER_IDS[x]},${NODE_IDS[n]},${CONTAINER_IDS[x]},${CONTAINER_IDS[x]}" >> containers.csv
			# Increments/Resets Node Index N and Container Index X (for evenly distributing containers across nodes)
			((x=x+1))
			((n=n+1))
			if [[ "$n" == "$NODE_COUNT" ]]
			then
			n=0
			fi
		done
	done
done




