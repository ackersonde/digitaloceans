#!/bin/bash
SERVERS="ubuntu@$CTX_MASTER_HOST ubuntu@$CTX_SLAVE_HOST ackersond@$CTX_BUILD_HOST"

# login to the master and run WAKE_ON_LAN on build host, wait 10 seconds and proceed
ssh -o StrictHostKeyChecking=no ubuntu@$CTX_MASTER_HOST "wakeonlan 2c:f0:5d:5e:84:43"

sleep 10

for i in $SERVERS
do
   ssh -o StrictHostKeyChecking=no $i \
      "sudo ufw allow from $NEW_SERVER_IPV6 to any port 22 && \
      sudo ufw --force delete \`sudo ufw status numbered | grep $OLD_SERVER_IPV6 | grep -o -E '[0-9]+' | head -1\`"
done

ssh ubuntu
