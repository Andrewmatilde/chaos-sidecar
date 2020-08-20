#!/bin/bash
envoy -c /etc/envoy/envoy.yaml > /tmp/envoy.log 2>&1 &
/scr/ctr  -nodeID sidecar > /tmp/ctr.log 2>&1 &
while [[ true ]]; do
	sleep 1
done