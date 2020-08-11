######################################################################
# Python script to create CSV Files for DNC Load Testing with Jmeter #
######################################################################

import csv
import sys
import os
import uuid
import math

path = "C:/Users/t-cobish/Desktop/Networking-Aquarius/src/dnc/test/jmeter/jmeter/"

if len(sys.argv) == 2:
    if sys.argv[1] == "-help":
        print("python generate.py networks subnets_per_network containers_per_subnet")

# Initializes variables with command line arguments if given
if len(sys.argv) == 5:
    net_count = int(sys.argv[1]) # Number of Networks to be Created
    subnet_count = int(sys.argv[2]) # Number of Subnets to be Joined per Network (Max 3)
    if subnet_count > 3:
        subnet_count = 3
    container_count = int(sys.argv[3]) # Number of Containers to be Created per Subnet
else:
    net_count = 2
    subnet_count = 2
    container_count = 2

# Total Number of Containers
container_total = net_count*subnet_count*container_count
# Max containers per node
containers_per_node = 3
# Number of nodes to be created
node_count = math.ceil(container_total/containers_per_node)

node_ids = []
net_ids = []
subnet_ids = []
container_ids = []

# Initializes ids

for i in range(1, node_count+1):
    node_ids.append("Node{}".format(uuid.uuid1()))

for i in range(1, net_count+1):
    net_ids.append(uuid.uuid1())

for i in range(1, subnet_count+1):
    subnet_ids.append("subnet{}".format(i))

for i in range(1, container_total+1):
    container_ids.append(uuid.uuid1())

# Clears out old CSV files
try:
    os.remove("nodes.csv")
    os.remove("nets.csv")
    os.remove("subnets.csv")
    os.remove("containers.csv")
except:
    pass

# Writes new CSV files
with open(path + 'nodes.csv', 'w', newline="") as csvfile:
    fieldnames = ['node_id', 'ip_address', 'orchestrator_type', 'location_type']
    writer = csv.DictWriter(csvfile, fieldnames=fieldnames, delimiter=",")
    for i in range(node_count):
        writer.writerow({'node_id' : node_ids[i], 'ip_address' : "127.0.0.1", 'orchestrator_type' : "Kubernetes", 'location_type': "dummmy_location"})

with open(path + 'nets.csv', 'w', newline="") as csvfile:
    fieldnames = ['net_id', 'net_type']
    writer = csv.DictWriter(csvfile, fieldnames=fieldnames, delimiter=",")
    for i in range(net_count):
        writer.writerow({'net_id' : net_ids[i], 'net_type' : "AzureNet"})

with open(path + 'subnets.csv', 'w', newline="") as csvfile:
    fieldnames = ['subnet_id', 'net_id']
    writer = csv.DictWriter(csvfile, fieldnames=fieldnames, delimiter=",")
    for i in range(net_count):
        for j in range(subnet_count):
            writer.writerow({'subnet_id': subnet_ids[j], 'net_id' : net_ids[i]})

with open(path + 'containers.csv', 'w', newline="") as csvfile:
    fieldnames = ['subnet_id', 'net_id', 'container_id', 'node_id', 'pod_name', 'pod_namespace']
    writer = csv.DictWriter(csvfile, fieldnames=fieldnames, delimiter=",")
    x = 0
    n = 0
    for i in range(net_count):
        for j in range(subnet_count):
            for k in range(container_count):
                writer.writerow({'subnet_id': subnet_ids[j], 'net_id' : net_ids[i], 'container_id' : container_ids[x], 'node_id' : node_ids[n], 'pod_name' : container_ids[x], "pod_namespace" : container_ids[x]})
                x += 1
                n += 1
                if n == node_count:
                    n = 0