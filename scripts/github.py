from nacl import encoding, public
from Crypto.PublicKey import RSA
from cryptography.hazmat.primitives import serialization
from pathlib import Path
from requests.exceptions import HTTPError
from time import time
from types import SimpleNamespace

import base64
import jwt
import os
import requests
import sys

GITHUB_APP_CLIENT_ID = os.environ['GITHUB_APP_CLIENT_ID']
GITHUB_INSTALL_ID = os.environ['GITHUB_INSTALL_ID']
GITHUB_SECRETS_PK_PEM = os.environ['GITHUB_SECRETS_PK_PEM']


def fatal(message):
    print('Error: {}'.format(message), file=sys.stderr)
    sys.exit(1)

# Token Exchange requires a JWT in the Auth Bearer header with this format
def generate_id_token(iss, expire_seconds=600):
    #raw_pem_key = open(GITHUB_SECRETS_PK_PEM_FILE, "r").read()
    signing_key = serialization.load_pem_private_key(GITHUB_SECRETS_PK_PEM.encode(), password=None)

    token = jwt.encode(
        {'iss': iss, 'iat': int(time()), 'exp': int(time()) + expire_seconds},
        signing_key, algorithm='RS256')

    key = RSA.import_key(GITHUB_SECRETS_PK_PEM)
    decoded = jwt.decode(token, key.public_key().export_key(), algorithms=['RS256'])
    if decoded['iss'] != iss:
        raise ValueError('Invalid token')

    return token


def encrypt(public_key: str, secret_value: str) -> str:
    """Encrypt a Unicode string using the public key."""
    public_key = public.PublicKey(public_key.encode("utf-8"), encoding.Base64Encoder())
    sealed_box = public.SealedBox(public_key)

    encrypted = sealed_box.encrypt(secret_value.encode("utf-8"))

    return base64.b64encode(encrypted).decode("utf-8")


def update_secret(token_headers: dict, github_JSON: dict, args: SimpleNamespace):
    secret_name = args.name
    b64encode = args.base64

    # call out B64 encoded secrets in string name
    if b64encode and not secret_name.endswith("_B64"):
        secret_name += "_B64"

    # atm, this script only supports Org secrets which I prefix with CTX_
    if not secret_name.startswith("CTX_"):
        secret_name = "CTX_" + secret_name

    payload = ""
    if args.filepath:
      file = Path(args.filepath)
      if b64encode:
        base64_bytes = base64.b64encode(file.read_bytes())
        payload = base64_bytes.decode("utf-8")
      else:
        payload = file.read_text()
    else:
      payload = args.value
      if b64encode:
        base64_bytes = base64.b64encode(bytes(payload, "utf-8"))
        payload = base64_bytes.decode("utf-8")

    secrets_url = 'https://api.github.com/orgs/ackersonde/actions/secrets'
    encrypted_value = encrypt(github_JSON['key'], payload)

    try:
      r = requests.put(
        f'{secrets_url}/{secret_name}',
        json={"encrypted_value": f"{encrypted_value}",
              "key_id": f"{github_JSON['key_id']}",
              "visibility": "all"},
        headers=token_headers)
      r.raise_for_status()

      print("Updated " + secret_name)
    except HTTPError as http_err:
      fatal(f'HTTP error occurred during secret update: {http_err}')
    except Exception as err:
      fatal(f'Other error occurred during secret update: {err}')

def fetch_token_headers():
    id_token = generate_id_token(iss=GITHUB_APP_CLIENT_ID)

    url = f'https://api.github.com/app/installations/{GITHUB_INSTALL_ID}/access_tokens'
    auth_headers = {'Accept': 'application/vnd.github.v3+json',
                    'Authorization': f'Bearer {id_token}'}
    resp = requests.post(url, headers=auth_headers)
    resp.raise_for_status()
    output = resp.json()

    token = output['token']
    return {'Accept': 'application/vnd.github.v3+json',
            'Authorization': f'token {token}'}

def fetch_public_key(token_headers: str):
    url = 'https://api.github.com/orgs/ackersonde/actions/secrets/public-key'

    resp = requests.get(url, headers=token_headers)
    resp.raise_for_status()
    return resp.json()
