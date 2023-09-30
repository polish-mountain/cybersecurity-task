#!/bin/env bash

set -e


avahi-browse -arp | while read -r line; do
    # split line into array
    IFS=';' read -ra DATA_ARR <<< "$line"
    dev_name=$(echo -e ${DATA_ARR[3]})
    echo -e ${DATA_ARR[3]}
    # jq -n \
    #     --arg name "${dev_name}" \
    #     --arg ip "${DATA_ARR[7]}" \
    #     --arg port "${DATA_ARR[8]}" \
    #     '{name: $name, ip: $ip, port: $port}'
done
```
