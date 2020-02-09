# A tiny CLI to manage AWS Credentials

`list` outputs a simple report to get a quick overview about credentials:

    AccessKeyId         	Age 	Status  	LastUsed                      	Username
    AKIAUGGSTKSE4EZDC3PO	1   	Inactive	never                         	IamRobot
    AKIAUGGSTKSE7KBIR2D2	38  	Active  	2019-10-22 07:02:00 +0000 UTC 	IamRobot

`all` outputs the same report with all credentials for the current account.

`new` creates a new pair of credentials and (optionally) writes them to `~/.aws/credentials` file.

All commands can be seen with `credentials -h`:

    Description:
        AWS Credential manager

    Sub-commands:
        credentials all       Displays a list of all keys for the current account
        credentials check     Check access key ages. Fail if any are older than set days
        credentials delete    Delete a key, shortcut: rm
        credentials disable   Disable (deactivate) a key
        credentials enable    Enable (activate) a key
        credentials list      List access keys, shortcut: ls
        credentials new       Create a new key


Run `-h` with subcommand to see usage:

    $ credentials new -h
    credentials new [--write] [--username=string]

    Description:
        Create a new key

    Options:
        -w, --write      Write new key to credentials file
        -u, --username   User to whom key will be created


    

## How to install?

    go get -v github.com/kerma/aws-credentials/cmd/credentials

