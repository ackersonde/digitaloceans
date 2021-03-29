#!/bin/bash
mkdir ~/.ssh
cat > ~/.ssh/id_rsa <<EOF
$CTX_GITHUB_SSH_DEPLOY_PRIVKEY
EOF

chmod 400 ~/.ssh/id_rsa
