#!/bin/bash

set -ex

HOST=teamvite.com

# Build on server
ssh $HOST bash -s <<'REMOTE'
    cd ~/teamvite
    git pull
    bin/build.sh
REMOTE

# Deploy binary
REMOTE_FILE=teamvite-$(ssh $HOST "md5sum ~/teamvite/teamvite | head -c8")
if ssh $HOST ls "/var/www/teamvite/$REMOTE_FILE" 2>/dev/null; then
    ssh $HOST touch "/var/www/teamvite/$REMOTE_FILE"
    echo "file exists, not copying"
else
    ssh $HOST "cp ~/teamvite/teamvite /var/www/teamvite/$REMOTE_FILE"
fi
ssh $HOST ln -sf "/var/www/teamvite/$REMOTE_FILE" /var/www/teamvite/teamvite

# Deploy secrets
echo "deploying config"
TMPCONFIG=$(mktemp)
trap "rm -f $TMPCONFIG" EXIT
sops -d secrets/config.json > "$TMPCONFIG"
scp "$TMPCONFIG" "root@$HOST:/var/www/teamvite/config.json"
ssh root@$HOST "chown throwingbones:throwingbones /var/www/teamvite/config.json && chmod 600 /var/www/teamvite/config.json"

echo "restarting service"
ssh root@$HOST systemctl restart teamvite

# keep the last 3 deploys
echo "removing old deploys"
ssh $HOST 'ls -t /var/www/teamvite/teamvite-* | tail -n +4 | xargs -I{} rm {}'

echo "uptime of service (should be larger than 3s and less than 10s)"
sleep 3
ssh $HOST systemctl status teamvite | grep -Po ".*; \K(.*)(?= ago)"
