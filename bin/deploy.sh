#!/bin/bash

REMOTE_FILE=teamvite-$(date +%F_%H-%M-%S)
scp teamvite root@ccrawa.org:/var/www/teamvite/$REMOTE_FILE
ssh root@ccrawa.org ln -sf /var/www/teamvite/$REMOTE_FILE /var/www/teamvite/teamvite
ssh root@ccrawa.org service teamvite restart
# keep the last 3 deploys of teamvite
ssh root@ccrawa.org 'find /var/www/teamvite/ -type f -name "teamvite-*" -print |sort -r | tail -n +4 |xargs ls'
ssh root@ccrawa.org 'find /var/www/teamvite/ -type f -name "teamvite-*" -print |sort -r | tail -n +4 |xargs rm'
ssh root@ccrawa.org service teamvite status
