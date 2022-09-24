#!/bin/bash

REMOTE_FILE=teamvite-$(date +%F_%H-%M-%S)
REMOTE_FILE=teamvite-$(md5sum teamvite |head -c8)
# check if file exists
if ssh teamvite.com ls "/var/www/teamvite/$REMOTE_FILE"; then
    ssh teamvite.com touch "/var/www/teamvite/$REMOTE_FILE"
    echo "file exists, not uploading"
else
    scp teamvite "teamvite.com:/var/www/teamvite/$REMOTE_FILE"
fi
ssh teamvite.com ln -sf "/var/www/teamvite/$REMOTE_FILE" /var/www/teamvite/teamvite
echo "restarting service"
ssh root@teamvite.com systemctl restart teamvite
# keep the last 3 deploys of teamvite
echo "removing old deploys"
ssh teamvite.com 'ls -t /var/www/teamvite/teamvite-* | tail -n +4 |xargs -I{} ls {}'
ssh teamvite.com 'ls -t /var/www/teamvite/teamvite-* | tail -n +4 |xargs -I{} rm {}'
echo "uptime of service (should be larger than 3s and less than 10s)"
sleep 3
ssh teamvite.com systemctl status teamvite | grep -Po ".*; \K(.*)(?= ago)"
# ssh teamvite.com service teamvite status
