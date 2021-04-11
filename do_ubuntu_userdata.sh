#!/bin/bash

# TODO: replace this key w/ 30 day rotation one
cat > /root/.ssh/id_rsa <<EOF
$CTX_RASPBERRYPI_SSH_PRIVKEY
EOF

chmod 400 /root/.ssh/id_rsa
touch ~/.hushlogin

# Setup Advanced DO monitoring
echo "deb https://repos.insights.digitalocean.com/apt/do-agent/ main main" > /etc/apt/sources.list.d/digitalocean-agent.list
curl -fsSL https://repos.insights.digitalocean.com/sonar-agent.asc | apt-key add -

apt-get update
apt-get -y install docker.io do-agent
apt-get --yes -o Dpkg::Options::="--force-confold" dist-upgrade

systemctl start docker
systemctl enable docker

# TODO: setup ipv6 capability?
#cat > /etc/docker/daemon.json <<EOF
# {
#  "ipv6": true,
#  "fixed-cidr-v6": "<DO_RANGE>:ffff::/80"
#}
#EOF
#systemctl restart docker

#ip6tables -t nat -A POSTROUTING -s <DO_RANGE>:ffff::/80 ! -o docker0 -j MASQUERADE
