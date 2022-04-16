#!/bin/bash

REMOTE_FILE=teamvite-$(date +%F_%H-%M-%S)
scp teamvite teamvite.com:/var/www/teamvite/$REMOTE_FILE
ssh teamvite.com ln -sf /var/www/teamvite/$REMOTE_FILE /var/www/teamvite/teamvite
ssh teamvite.com systemctl --user restart teamvite
# keep the last 3 deploys of teamvite
ssh teamvite.com 'find /var/www/teamvite/ -type f -name "teamvite-*" -print |sort -r | tail -n +4 |xargs ls'
ssh teamvite.com 'find /var/www/teamvite/ -type f -name "teamvite-*" -print |sort -r | tail -n +4 |xargs rm'
ssh teamvite.com service teamvite status
