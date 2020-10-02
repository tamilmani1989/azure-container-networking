#!/bin/bash

usage="usage: $0 network_count subnets_per_network containers_per_subnet containers_per_node (secondary_ips_per_container)"

default_secondary_ip_count=0

# Read from Command Line
NET_COUNT=${1?$usage}
SUBNETS_PER_NET=${2?$usage}
CONTAINERS_PER_SUBNET=${3?$usage}
CONTAINERS_PER_NODE=${4?$usage}
SECONDARY_IPS_PER_CONTAINER=${5:-$default_secondary_ip_count}

# Ensure Limits Honored
if [[ SUBNETS_PER_NET -gt 3 ]]
then
  echo "Maximum 3 Subnets Per Network"
  SUBNETS_PER_NET=3
fi

if [[ CONTAINERS_PER_SUBNET -gt 1000000 ]]
then
  echo "Maximum 1 million Containers Per Subnet"
  CONTAINERS_PER_SUBNET=1000000
fi

if [[ CONTAINERS_PER_NODE -gt 3 ]]
then
  echo "Maximum 3 Containers Per Node"
  CONTAINERS_PER_NODE=3
fi

if [[ IPS_PER_CONTAINER -gt 250 ]]
then
  echo "Maximum 250 Secondary IPs Per Container"
  IPS_PER_CONTAINER=250
fi

CONTAINER_COUNT=$((NET_COUNT * SUBNETS_PER_NET * CONTAINERS_PER_SUBNET))
NODE_COUNT=$((CONTAINER_COUNT / CONTAINERS_PER_NODE + 1))

echo "Creating Test Plan with ${NET_COUNT} Networks, ${SUBNETS_PER_NET} Subnets Per Network, ${CONTAINERS_PER_SUBNET} Containers Per Subnet, and ${CONTAINERS_PER_NODE} Containers Per Node."

function genuuid() {
    od -x /dev/urandom | head -1 | awk '{OFS="-"; print $2$3,$4,$5,$6,$7$8$9}'
}

# Initialize Node IDs
for ((i=0; i<NODE_COUNT; i++ ))
do
	NODE_IDS+=("$(genuuid)")
done

# Initialize Subnet IDs
# starting at 1, since mock nmagent only responds to subnet1, subnet2, subnet3
for ((i=1; i<=SUBNETS_PER_NET; i++ ))
do
	SUBNET_IDS+=("subnet${i}")
done

# Initialize Net IDs
for ((i=0; i<NET_COUNT; i++ ))
do
	NET_IDS+=("$(genuuid)")
done

# Initialize Container IDs
for ((i=0; i<CONTAINER_COUNT; i++ ))
do
	CONTAINER_IDS+=("$(genuuid)")
done

# Clear out old CSV Files
rm -f nets.csv subnets.csv containers.csv nodes.csv

# Create New CSV Files
# nodes.csv (node_id, ip_address, orchestrator_type, location_type)
for (( i=0; i<NODE_COUNT; i++ ))
do
	echo "${NODE_IDS[i]},127.0.0.1,ServiceFabric,dummy_location" >> nodes.csv
done

# nets.csv (net_id, net_type)
for (( i=0; i<NET_COUNT; i++ ))
do
	echo "${NET_IDS[i]},AzureNet" >> nets.csv
done

# subnets.csv (subnet_id, net_id)
for (( i=0; i<NET_COUNT; i++ ))
do
	for (( j=0; j<SUBNETS_PER_NET; j++ ))
	do
		echo "${SUBNET_IDS[j]},${NET_IDS[i]}" >> subnets.csv
	done
done

# containers.csv (subnet_id, net_id, container_id, node_id, pod_name, pod_namespace)
x=0
n=0
for (( i=0; i<NET_COUNT; i++ ))
do
	for (( j=0; j<SUBNETS_PER_NET; j++ ))
	do
		for (( k=0; k<CONTAINERS_PER_SUBNET; k++ ))
		do
			echo "${SUBNET_IDS[j]},${NET_IDS[i]},${CONTAINER_IDS[x]},${NODE_IDS[n]},pod_name_${CONTAINER_IDS[x]},pod_namespace_${CONTAINER_IDS[x]},${SECONDARY_IPS_PER_CONTAINER}" >> containers.csv
			# Increments/Resets Node Index N and Container Index X (for evenly distributing containers across nodes)
			((x++))
			((n++))
			if [[ "$n" == "$NODE_COUNT" ]]
			then
			n=0
			fi
		done
	done
done
