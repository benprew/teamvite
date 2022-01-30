#!/bin/bash

REMOTE_FILE=teamvite-$(date +%F_%H-%M-%S)
scp teamvite root@ccrawa.org:/var/www/teamvite/$REMOTE_FILE
ssh root@ccrawa.org ln -sf /var/www/teamvite/$REMOTE_FILE /var/www/teamvite/teamvite
ssh root@ccrawa.org service teamvite restart
ssh root@ccrawa.org 'find /var/www/teamvite/ -type f -mtime +3 -name "teamvite-*" -delete -print'
