#!/bin/bash
SERVERS="ubuntu@$CTX_MASTER_HOST ubuntu@$CTX_SLAVE_HOST ackersond@$CTX_BUILD_HOST"

# login to the master and run WAKE_ON_LAN on build host, wait 10 seconds and proceed
ssh -o StrictHostKeyChecking=no ubuntu@$CTX_MASTER_HOST "wakeonlan 2c:f0:5d:5e:84:43"

sleep 10

for i in $SERVERS
do
   ssh -o StrictHostKeyChecking=no $i "\
      sudo ip6tables -A INPUT -s $NEW_SERVER_IPV6 -m conntrack -p tcp --dport 22 --ctstate NEW,ESTABLISHED -j ACCEPT && \
      sudo ip6tables -D INPUT -s $OLD_SERVER_IPV6 -m conntrack -p tcp --dport 22 --ctstate NEW,ESTABLISHED -j ACCEPT && \
      sudo ip6tables-save -f /etc/iptables/rules.v6"
done
