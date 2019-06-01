#!/bin/sh
public_ip_address=$(curl -s https://checkip.amazonaws.com)
json='{"inbound_rules":[{"protocol":"tcp","ports":"22","sources":{"addresses":["'"$public_ip_address"'"]}}]}'

curl -X POST -H "Content-Type: application/json" -H "Authorization: Bearer $CTX_DIGITALOCEAN_TOKEN" \
-d $json "https://api.digitalocean.com/v2/firewalls/$CTX_DIGITALOCEAN_FIREWALL/rules"
