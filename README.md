# A tiny CLI to manage AWS Credentials

`list` outputs a simple report to get a quick overview about credentials:

    AccessKeyId         	Age 	Status  	LastUsed                      	Username
    AKIAUGGSTKSE4EZDC3PO	1   	Inactive	never                         	IamRobot
    AKIAUGGSTKSE7KBIR2D2	38  	Active  	2019-10-22 07:02:00 +0000 UTC 	IamRobot

`all` outputs the same report with all credentials for the current account.

`new` creates a new pair of credentials and (optionally) writes them to `~/.aws/credentials` file.

`enable`, `disable` and `delete`... well, guess?

## How to install?

    go get -v github.com/kerma/aws-credentials/cmd/credentials