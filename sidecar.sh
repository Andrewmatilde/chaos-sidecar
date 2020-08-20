#!/bin/bash
envoy -c /etc/envoy/envoy.yaml > /dev/null 2>&1 &
/scr/ctr  -nodeID sidecar > /dev/null 2>&1 &
while [[ true ]]; do
	sleep 1
done