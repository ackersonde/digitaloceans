#!/usr/bin/env python3
from requests.exceptions import HTTPError
import argparse
import vault  # found over in pi-ops repo
import sys


def main():
    parser = argparse.ArgumentParser(description="Update Vault secret")
    parser.add_argument(
        "-n",
        "--name",
        type=str,
        dest="name",
        help="secret name to update",
        required=True,
    )
    parser.add_argument(
        "-f",
        "--filepath",
        type=str,
        dest="filepath",
        help="secret value is the contents of this file",
    )
    args = parser.parse_args()

    try:
        vault.update_secret(args)
    except HTTPError as http_err:
        print("HTTP Error: {}".format(http_err), file=sys.stderr)
    except Exception as err:
        print("Other Error: {}".format(err), file=sys.stderr)


if __name__ == "__main__":
    main()
