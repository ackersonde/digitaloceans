#!/bin/bash
export doSSHPubKey=$(echo $encodedDOSSHLoginPubKey | base64 --decode)
export circleCIDeployPubKey=$(echo $encodedCircleCIDeployPubKey | base64 --decode)
export consolePasswdHash=$(echo $encodedConsolePasswdHash | base64 --decode)
sed -i -e "s@{{login_ssh_pubkey}}@$doSSHPubKey@" digitalocean_ignition.json
sed -i -e "s@{{circleci_deploy_pubkey}}@$circleCIDeployPubKey@" digitalocean_ignition.json
sed -i -e "s/{{console_passwd_hash}}/$consolePasswdHash/" digitalocean_ignition.json
sed -i -e "s/{{deploy_user}}/$deployUser/" sshd_config
export encodedSSHDConfig=$(base64 -w 0 sshd_config)
sed -i -e "s/{{deploy_user}}/$deployUser/g" digitalocean_ignition.json
sed -i -e "s/{{encoded_sshd_config}}/$encodedSSHDConfig/" digitalocean_ignition.json
