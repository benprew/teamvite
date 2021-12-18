#!/bin/bash

set -x

REMOTE_FILE=teamvite-$(date +%F)
scp teamvite root@ccrawa.org:/var/www/teamvite/$REMOTE_FILE
ssh root@ccrawa.org ln -sf /var/www/teamvite/$REMOTE_FILE /var/www/teamvite/teamvite
ssh root@ccrawa.org service teamvite restart