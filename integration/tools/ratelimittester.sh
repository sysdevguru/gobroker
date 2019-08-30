#!/bin/sh

APCA_API_KEY_ID=AKJIQEC9UKRKLZ6W5N15
APCA_API_SECRET_KEY=bvS1TZdpA/yp2YG2sT27MW8debwU3BGDVIigOAyn

for i in {1..300}
do
    curl http://127.0.0.1/api/v1/accounts -H "APCA-API-KEY-ID: $APCA_API_KEY_ID" -H "APCA-API-SECRET-KEY: $APCA_API_SECRET_KEY" -sf > /dev/null
    if [ $? -gt 0 ]
    then
        echo $i && exit
    fi
done
