![Deploy DigitalOcean Droplet](https://github.com/ackersonde/digitaloceans/workflows/Deploy%20DigitalOcean%20Droplet/badge.svg)

# DigitalOceans
Since Vodafone's DS-Lite-Tunnel doesn't offer native IPv4 addresses (and many services incl. [Github Actions](https://github.com/actions/virtual-environments/issues/668) & the [Slack API](https://api.slack.com/authentication/best-practices#ip_allowlisting) don't speak IPv6 yet), I had to move my [homepage](https://ackerson.de), [slack](https://github.com/ackersonde/bender-slackbot) & [telegram](https://github.com/ackersonde/telegram-bot) bots off my Raspberry Pi infrastructure and back to Digital Ocean.

# Build & Deploy [DigitalOcean Droplet](https://cloud.digitalocean.com/droplets)
Using the golang api from [godo](https://github.com/digitalocean/godo), every push to this repository creates a [custom](https://github.com/ackersonde/digitaloceans/blob/main/scripts/do_ubuntu_userdata.sh) Ubuntu <img src="https://assets.ubuntu.com/v1/29985a98-ubuntu-logo32.png" width="16"> droplet in FRA1.

# Automated Deployment
I have a [weekly cronjob](https://github.com/ackersonde/pi-ops/blob/master/scripts/crontab.txt) running on one of my raspberry PIs which triggers this deployment after regenerating the SSL certificate ([only valid for 10d](https://github.com/ackersonde/pi-ops/blob/master/scripts/gen_new_deploy_keys.sh#L18)) required by the various servers.

# WARNING
Every push to this repo will result in a new Droplet created at DigitalOcean => +$7 / month, tearing down and redeploying websites and bots while also updating DNS entries for *.ackerson.de.

Use git commit msg string snippet `[skip ci]` to avoid this.
