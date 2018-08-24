#!/bin/sh
i=0
while [ $i -lt 5 ]
do
    apt-get -y update >> install.log  2>&1
    apt-get install -y ebtables >> install.log  2>&1
    if [ "$?" -eq 0 ]; then
        break
    fi
    i=$(($i+1))
    sleep 10
done
exit 0
