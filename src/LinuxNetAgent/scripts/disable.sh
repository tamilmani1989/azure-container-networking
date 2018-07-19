#!/bin/sh
./servicetunnel -mode=disable > servicetunnel.log
status=$?
exit $status