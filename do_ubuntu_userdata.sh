#!/bin/bash

# TODO: replace this key w/ 30 day rotation one
cat > /root/.ssh/id_rsa <<EOF
$CTX_RASPBERRYPI_SSH_PRIVKEY
EOF

chmod 400 /root/.ssh/id_rsa
touch ~/.hushlogin

# Setup Advanced DO monitoring
curl -fsSL https://repos.insights.digitalocean.com/sonar-agent.asc | gpg --dearmor | tee /usr/share/keyrings/sonar-agent-keyring.gpg
add-apt-repository "deb https://repos.insights.digitalocean.com/apt/do-agent/ main main"

# Install docker engine
#apt update && apt install apt-transport-https ca-certificates curl software-properties-common
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor | tee /usr/share/keyrings/docker-keyring.gpg
add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"

apt-get update && apt-get install -y docker-ce do-agent
apt-get upgrade -y

# TODO: setup ipv6 capability?
#cat > /etc/docker/daemon.json <<EOF
# {
#  "ipv6": true,
#  "fixed-cidr-v6": "<DO_RANGE>:ffff::/80"
#}
#EOF
#systemctl restart docker

#ip6tables -t nat -A POSTROUTING -s <DO_RANGE>:ffff::/80 ! -o docker0 -j MASQUERADE
