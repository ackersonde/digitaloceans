#!/usr/bin/env python3
from requests.exceptions import HTTPError
import argparse
import github   # found over in pi-ops repo


def main():
    parser = argparse.ArgumentParser(description='Update Github organization secret')
    parser.add_argument('-n', '--name', type=str, dest='name', help='secret name to update', required=True)
    parser.add_argument('-v', '--value', type=str, dest='value', help='update secret to this value')
    parser.add_argument('-f', '--filepath', type=str, dest='filepath', help='update secret to the contents of this file')
    parser.add_argument('-b', '--storeB64', type=bool, dest='base64', default=False, required=False, help='store value as b64 encoded')

    args = parser.parse_args()
    if not args.value and not args.filepath:
        github.fatal('Please provide either a secret `value` or `filepath`')

    # https://docs.github.com/en/free-pro-team@latest/rest/reference/actions#secrets
    try:
        token_headers = github.fetch_token_headers()
        github_public_key = github.fetch_public_key(token_headers)
        github.update_secret(token_headers, github_public_key, args)
    except HTTPError as http_err:
        github.fatal(f'HTTP error occurred during auth: {http_err}')
    except Exception as err:
        github.fatal(f'Other error occurred during auth: {err}')


if __name__ == '__main__':
    main()
