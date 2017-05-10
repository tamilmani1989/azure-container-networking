#!/bin/bash
ret=0
error=$(tail -1 valgrind.log | head -1|cut -d" " -f4)
cat valgrind.log
if [ $error != $ret ]; then
  echo "error exist"
  exit 2
fi