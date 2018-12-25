curl -X POST -H "Content-Type: application/json" -H "Authorization: Bearer $digitalOceanToken" \
-d '{"inbound_rules":[{"protocol":"tcp","ports":"22","sources":{"addresses":"$public_ip_address"}}]}' \
"https://api.digitalocean.com/v2/firewalls/$doFirewallID/rules"
