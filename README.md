![Deploy DigitalOcean Droplet](https://github.com/ackersonde/digitaloceans/workflows/Deploy%20DigitalOcean%20Droplet/badge.svg)

# DigitalOceans
Since Vodafone's DS-Lite-Tunnel doesn't offer native IPv4 addresses (and many services incl. Github Actions & the Slack API don't speak IPv6 yet), I had to move my [websites](https://ackerson.de) and [bots](https://github.com/ackersonde/bender-slackbot) out of my home, PI infrastructure and back to Digital Ocean.

# Build & Deploy [DigitalOcean Droplet](https://cloud.digitalocean.com/droplets)
Using the golang api from [godo](https://github.com/digitalocean/godo), every push to this repository creates a [custom](https://github.com/ackersonde/digitaloceans/blob/main/scripts/do_ubuntu_userdata.sh) Ubuntu <img src="https://assets.ubuntu.com/v1/29985a98-ubuntu-logo32.png" width="16"> droplet in FRA1.

# Automated Deployment
I have a [monthly cronjob](https://github.com/ackersonde/pi-ops/blob/master/scripts/crontab.txt) running on one of my raspberry PIs which triggers this deployment after regenerating the SSL certificate (only valid for 5 weeks) required by the various servers.

# WARNING
Every push to this repo will result in a new Droplet created at DigitalOcean => +$5 / month, tearing down and redeploying websites and bots while also updating DNS entries for *.ackerson.de.

Use git commit msg string snippet `[skip ci]` to avoid this.
