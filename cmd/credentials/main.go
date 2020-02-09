package main

import (
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	credentials "github.com/kerma/aws-credentials"
	"github.com/teris-io/cli"
)

func main() {

	// setup aws session
	sess := session.Must(session.NewSession(&aws.Config{
		MaxRetries: aws.Int(3),
	}))
	c := credentials.New(iam.New(sess, &aws.Config{}))

	// setup commands
	list := cli.NewCommand("list", "List access keys").
		WithShortcut("ls").
		WithOption(cli.NewOption("max-age", "Maximum age for the key").WithChar('m').WithType(cli.TypeInt)).
	  	WithOption(cli.NewOption("username", "Username for the query").WithChar('u').WithType(cli.TypeString)).
  		WithAction(func(args []string, options map[string]string) int {
			if keyMaxAge, ok := options["max-age"]; ok {
				c.KeyMaxAge, _ = strconv.Atoi(keyMaxAge)
			}
			if username, ok := options["username"]; ok {
				return c.RunUserListCmd(username)
			}
			return c.RunListCmd()
  		})
	all := cli.NewCommand("all", "Displays a list of all keys for the current account").
		WithOption(cli.NewOption("max-age", "Maximum age for the key").WithChar('m').WithType(cli.TypeInt)).
		WithAction(func(args []string, options map[string]string) int {
			if keyMaxAge, ok := options["max-age"]; ok {
				c.KeyMaxAge, _ = strconv.Atoi(keyMaxAge)
			}
			return c.RunAllCmd()
		})
	check := cli.NewCommand("check", "Check access key ages. Fail if any are older than set days").
		WithOption(cli.NewOption("max-age", "Maximum age for the key").WithChar('m').WithType(cli.TypeInt)).
		WithOption(cli.NewOption("all", "Check all keys for the current account").WithChar('a').WithType(cli.TypeBool)).
		WithAction(func(args []string, options map[string]string) int {
			if keyMaxAge, ok := options["max-age"]; ok {
				c.KeyMaxAge, _ = strconv.Atoi(keyMaxAge)
			}
			if _, ok := options["all"]; ok {
				return c.RunCheckAllKeys()
			}
			return c.RunCheckKeys()
		})
	deleteCmd := cli.NewCommand("delete", "Delete a key").
		WithShortcut("rm").
		WithArg(cli.NewArg("key-id", "Access key ID")).
		WithOption(cli.NewOption("username",
			"Username is required when deleting a key which does not belong to the current user").
			WithChar('u').WithType(cli.TypeString)).
		WithAction(func(args []string, options map[string]string) int {
			if username, ok := options["username"]; ok {
				return c.RunDeleteUserKeyCmd(args[0], username)
			}
			return c.RunDeleteCmd(args[0])
		})
	disable := cli.NewCommand("disable", "Disable (deactivate) a key").
		WithArg(cli.NewArg("keyId", "Access key ID")).
		WithOption(cli.NewOption("username",
			"Username is required when disabling a key which does not belong to the current user").
			WithChar('u').WithType(cli.TypeString)).
		WithAction(func(args []string, options map[string]string) int {
			if username, ok := options["username"]; ok {
				return c.RunDisableUserKeyCmd(args[0], username)
			}
			return c.RunDisableCmd(args[0])
	})
	enable := cli.NewCommand("enable", "Enable (activate) a key").
		WithArg(cli.NewArg("keyId", "Access key ID")).
		WithOption(cli.NewOption("username",
			"Username is required when activating a key which does not belong to the current user").
			WithChar('u').WithType(cli.TypeString)).
		WithAction(func(args []string, options map[string]string) int {
			if username, ok := options["username"]; ok {
				return c.RunEnableUserKeyCmd(args[0], username)
			}
			return c.RunEnableCmd(args[0])
		})
	newCmd := cli.NewCommand("new", "Create a new key").
		WithOption(cli.NewOption("write", "Write new key to credentials file").WithChar('w').WithType(cli.TypeBool)).
		WithOption(cli.NewOption("username", "User to whom key will be created").WithChar('u').WithType(cli.TypeString)).
		WithAction(func(args []string, options map[string]string) int {
			if _, ok := options["write"]; ok {
				c.WriteCredentialsFile = true
			}
			if username, ok := options["username"]; ok {
				return c.RunUserNewCmd(username)
			}
			return c.RunNewCmd()
		})

	// setup app
	app := cli.New("AWS Credential manager").
		WithCommand(all).
		WithCommand(check).
		WithCommand(deleteCmd).
		WithCommand(disable).
		WithCommand(enable).
		WithCommand(list).
		WithCommand(newCmd).
		WithAction(func(args []string, options map[string]string) int {
			return c.RunListCmd()  // run list as a default command
		})

	os.Exit(app.Run(os.Args, os.Stdout))
}

