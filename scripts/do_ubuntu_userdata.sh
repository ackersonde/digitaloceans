#!/bin/bash
echo -n "$SERVER_DEPLOY_CACERT_B64" | base64 -d | tee /root/.ssh/id_ed25519-cert.pub
chmod 400 /root/.ssh/id_ed25519-cert.pub
echo -n "$SERVER_DEPLOY_SECRET_B64" | base64 -d | tee /root/.ssh/id_ed25519
chmod 400 /root/.ssh/id_ed25519
echo -n "$SERVER_DEPLOY_PUBLIC_B64" | base64 -d | tee -a /root/.ssh/authorized_keys
echo -n "$TITAN_PUBLIC_KEY" | tee -a /root/.ssh/authorized_keys
echo -n "$ACME_JSON_B64" | base64 -d | tee /root/traefik/acme.json
chmod 600 /root/traefik/acme.json

# Setup Syncthing config
mkdir -p /root/syncthing/config /root/syncthing/2086h-4d0t2
echo ".trashed-*" > /root/syncthing/2086h-4d0t2/.stignore
echo "*.part" >> /root/syncthing/2086h-4d0t2/.stignore
chmod 600 /root/syncthing/2086h-4d0t2/.stignore
echo -n "$SYNCTHING_CONFIG_B64" | base64 -d | tee /root/syncthing/config/config.xml
chmod 600 /root/syncthing/config/config.xml
cat <<EOF > /root/syncthing/config/key.pem
$SYNCTHING_KEY
EOF
chmod 600 /root/syncthing/config/key.pem
cat <<EOF > /root/syncthing/config/cert.pem
$SYNCTHING_CERT
EOF
chmod 644 /root/syncthing/config/cert.pem
chown -R 1000:1000 /root/syncthing

touch ~/.hushlogin

# Setup Advanced DO monitoring
curl -fsSL https://repos.insights.digitalocean.com/sonar-agent.asc | gpg --dearmour -o /usr/share/keyrings/digitalocean-agent.gpg
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/digitalocean-agent.gpg] https://repos.insights.digitalocean.com/apt/do-agent/ main main" > /etc/apt/sources.list.d/digitalocean-agent.list

# prepare iptables persistence and unattended-upgrades install settings
debconf-set-selections <<EOF
iptables-persistent iptables-persistent/autosave_v4 boolean true
iptables-persistent iptables-persistent/autosave_v6 boolean true
unattended-upgrades unattended-upgrades/enable_auto_updates boolean true
EOF

# allow docker containers to talk to the internet
ip6tables -t nat -A POSTROUTING -s fd00::/80 ! -o docker0 -j MASQUERADE
dpkg-reconfigure -f noninteractive unattended-upgrades

apt-get -y remove docker docker-engine docker.io containerd runc
apt-get update
apt-get -y install wireguard ca-certificates curl gnupg lsb-release iptables-persistent do-agent

cat > /etc/wireguard/wg.conf << EOF
[Interface]
Address = 10.9.0.1/24,fd42:42:42::1/64
ListenPort = {{WG_DO_HOME_PORT}}
PrivateKey = {{WG_DO_PRIVATE_KEY}}

#pixel6
[Peer]
PublicKey = {{WG_HOME_PUBLIC_KEY}}
AllowedIPs = 10.9.0.2/32,fd42:42:42::2/128
PresharedKey = {{WG_DO_HOME_PRESHAREDKEY}}
EOF

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update
apt-get -y install docker-ce docker-ce-cli containerd.io

systemctl start docker
systemctl enable docker

# setup ipv6 capability in docker
cat > /etc/docker/daemon.json <<EOF
{
  "ipv6": true,
  "fixed-cidr-v6": "fd00::/80"
}
EOF
systemctl restart docker

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
Unattended-Upgrade::Automatic-Reboot-Time "05:00";
EOF

/usr/bin/wg-quick up wg
systemctl enable wg-quick@wg.service
