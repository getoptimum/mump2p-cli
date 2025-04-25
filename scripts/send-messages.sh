#!/bin/bash

for i in $(seq 1 100); do
  msg1=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 1000)
  msg2=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 1000)
  msg="${msg1}${msg2}${msg1}"   
  echo $i ${#msg}  
  dist/mump2p-linux publish --topic=customtopic --message="msg $i: $msg"
  sleep 0.125
done
