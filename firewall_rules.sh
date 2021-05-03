#!/bin/bash

if [[ $NEW_SERVER_IPV6 =~ ^192\.168\..* || $OLD_SERVER_IPV6 =~ ^192\.168\..* ]]
then
   echo "SERVER_IPV6 values are IPV4? Wisely refusing to cut local network access..."
   exit 0
else
   SERVERS="ubuntu@$CTX_MASTER_HOST ubuntu@$CTX_SLAVE_HOST ackersond@$CTX_BUILD_HOST"
   for i in $SERVERS
   do
      ssh $i \
         "sudo ufw allow from $NEW_SERVER_IPV6 to any port 22 && \
         sudo ufw --force delete \`sudo ufw status numbered | grep $OLD_SERVER_IPV6 | grep -o -E '[0-9]+' | head -1\`"
   done
fi
