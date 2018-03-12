#!/bin/bash
export doSSHPubKey=$(echo $encodedDOSSHLoginPubKey | base64 -d)
export circleCIDeployPubKey=$(echo $encodedCircleCIDeployPubKey | base64 -d)
export consolePasswdHash=$(echo $encodedConsolePasswdHash | base64 -d)

sed -i -e "s@{{login_ssh_pubkey}}@$doSSHPubKey@" digitalocean_ignition.json
sed -i -e "s@{{circleci_deploy_pubkey}}@$circleCIDeployPubKey@" digitalocean_ignition.json
sed -i -e "s@{{console_passwd_hash}}@$consolePasswdHash@" digitalocean_ignition.json
sed -i -e "s@{{deploy_user}}@$deployUser@" sshd_config

base64 sshd_config > out
export encodedSSHDConfig=$(tr -d '\n' < out)

sed -i -e "s@{{deploy_user}}@$deployUser@g" digitalocean_ignition.json
sed -i -e "s@{{encoded_sshd_config}}@$encodedSSHDConfig@" digitalocean_ignition.json
