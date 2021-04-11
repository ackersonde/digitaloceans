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

systemctl start docker
systemctl enable docker

echo unattended-upgrades unattended-upgrades/enable_auto_updates boolean true | debconf-set-selections
dpkg-reconfigure -f noninteractive unattended-upgrades

cat > /etc/apt/apt.conf.d/50unattended-upgrades << EOF
// Automatically upgrade packages from these (origin, archive) pairs
Unattended-Upgrade::Allowed-Origins {
    // ${distro_id} and ${distro_codename} will be automatically expanded
    "${distro_id} stable";
    "${distro_id} ${distro_codename}-security";
    "${distro_id} ${distro_codename}-updates";
//  "${distro_id} ${distro_codename}-proposed-updates";
};

// Do automatic removal of new unused dependencies after the upgrade
// (equivalent to apt-get autoremove)
Unattended-Upgrade::Remove-Unused-Dependencies "true";

// Automatically reboot *WITHOUT CONFIRMATION* if a
// the file /var/run/reboot-required is found after the upgrade
Unattended-Upgrade::Automatic-Reboot "true";
EOF


# TODO: setup ipv6 capability?
#cat > /etc/docker/daemon.json <<EOF
# {
#  "ipv6": true,
#  "fixed-cidr-v6": "<DO_RANGE>:ffff::/80"
#}
#EOF
#systemctl restart docker

#ip6tables -t nat -A POSTROUTING -s <DO_RANGE>:ffff::/80 ! -o docker0 -j MASQUERADE
