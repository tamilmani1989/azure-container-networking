#!/bin/sh
./servicetunnel -mode=enable > servicetunnel.log
status=$?
exit $status
