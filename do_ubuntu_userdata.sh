#!/bin/bash
echo -n "$CTX_SERVER_DEPLOY_CACERT_B64" | base64 -d | tee /root/.ssh/id_ed25519-cert.pub
chmod 400 /root/.ssh/id_ed25519-cert.pub
echo -n "$CTX_SERVER_DEPLOY_SECRET_B64" | base64 -d | tee /root/.ssh/id_ed25519
chmod 400 /root/.ssh/id_ed25519
echo -n "$CTX_SERVER_DEPLOY_PUBLIC_B64" | base64 -d | tee -a /root/.ssh/authorized_keys

mkdir /root/traefik
cat <<EOF >/root/traefik/acme.json
$ACME_JSON
EOF
chmod 600 /root/traefik/acme.json
touch ~/.hushlogin

# Setup Advanced DO monitoring
echo "deb https://repos.insights.digitalocean.com/apt/do-agent/ main main" > /etc/apt/sources.list.d/digitalocean-agent.list
curl -fsSL https://repos.insights.digitalocean.com/sonar-agent.asc | apt-key add -

apt-get update
apt-get -y install docker.io do-agent netfilter-persistent

systemctl start docker
systemctl enable docker

echo unattended-upgrades unattended-upgrades/enable_auto_updates boolean true | debconf-set-selections
dpkg-reconfigure -f noninteractive unattended-upgrades

cat > /etc/apt/apt.conf.d/50unattended-upgrades << EOF
Unattended-Upgrade::Allowed-Origins {
    "\${distro_id} stable";
    "\${distro_id} \${distro_codename}-security";
    "\${distro_id} \${distro_codename}-updates";
};

// Do automatic removal of new unused dependencies after the upgrade
// (equivalent to apt-get autoremove)
Unattended-Upgrade::Remove-Unused-Dependencies "true";

// Automatically reboot *WITHOUT CONFIRMATION* if a
// the file /var/run/reboot-required is found after the upgrade
Unattended-Upgrade::Automatic-Reboot "true";
EOF


# setup ipv6 capability
cat > /etc/docker/daemon.json <<EOF
{
  "ipv6": true,
  "fixed-cidr-v6": "fd00::/80"
}
EOF
systemctl restart docker

ip6tables -t nat -A POSTROUTING -s fd00::/80 ! -o docker0 -j MASQUERADE
mkdir /etc/iptables
netfilter-persistent save
