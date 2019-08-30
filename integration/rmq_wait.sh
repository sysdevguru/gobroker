#!/bin/bash

for i in `seq 1 30`;
do
    make ping -C integration/dockerfiles/rmq && exit 0
    sleep 1
done
echo Failed waiting for rabbitmq && exit 1