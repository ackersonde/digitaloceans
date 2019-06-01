#!/bin/sh

# Store current Firewall
curl -s -X GET -H "Content-Type: application/json" -H "Authorization: Bearer $CTX_DIGITALOCEAN_TOKEN" "https://api.digitalocean.com/v2/firewalls/$CTX_DIGITALOCEAN_FIREWALL" > doFirewall.json

# Because DigitalOcean is picky
sed -i -e "s/ports\":\"0/ports\":\"1-65535/g" doFirewall.json

# Recreate Firewall rules without Droplet
echo '{"name":' > disableFW.json
cat doFirewall.json | jq '.firewall.name' >> disableFW.json && echo ', "inbound_rules":' >> disableFW.json
cat doFirewall.json | jq '.firewall.inbound_rules' >> disableFW.json && echo ', "outbound_rules":' >> disableFW.json
cat doFirewall.json | jq '.firewall.outbound_rules' >> disableFW.json
echo "}" >> disableFW.json

# Apply disabledFW rules
curl -X PUT -H "Content-Type: application/json" -d @disableFW.json -H "Authorization: Bearer $CTX_DIGITALOCEAN_TOKEN" "https://api.digitalocean.com/v2/firewalls/$CTX_DIGITALOCEAN_FIREWALL"
