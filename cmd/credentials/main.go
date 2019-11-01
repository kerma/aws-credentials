package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	credentials "github.com/kerma/aws-credentials"
)

var (
	keyMaxAge int
	username  string
	writeFile bool
)

func main() {

	// command line flags setup
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listCmd.IntVar(&keyMaxAge, "m", credentials.DefaultKeyMaxAge, "Maximum access key age in days")
	listCmd.StringVar(&username, "u", "", "Username for the query")
	listCmd.StringVar(&username, "username", "", "Username for the query")

	allCmd := flag.NewFlagSet("all", flag.ExitOnError)
	allCmd.IntVar(&keyMaxAge, "m", credentials.DefaultKeyMaxAge, "Maximum access key age in days")

	newCmd := flag.NewFlagSet("new", flag.ExitOnError)
	newCmd.BoolVar(&writeFile, "w", false, "Write new key to credentials file")
	newCmd.StringVar(&username, "u", "", "Username for the query")

	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteCmd.StringVar(&username, "u", "", "Username (Required when deleting key that does not belong to current user)")
	deleteCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  credentials delete [-u <username>] <access-key-id>\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Username is required when deleting a key that does not belong to current user\n")
		os.Exit(1)
	}

	disableCmd := flag.NewFlagSet("disable", flag.ExitOnError)
	disableCmd.StringVar(&username, "u", "", "Username (Required when deleting key that does not belong to current user)")
	disableCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  credentials disable [-u <username>] <access-key-id>\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Username is required when disabling a key that does not belong to current user\n")
		os.Exit(1)
	}

	enableCmd := flag.NewFlagSet("enable", flag.ExitOnError)
	enableCmd.StringVar(&username, "u", "", "Username (Required when deleting key that does not belong to current user)")
	enableCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  credentials enable [-u <username>] <access-key-id>\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Username is required when enabling a key that does not belong to current user\n")
		os.Exit(1)
	}

	// session setup
	sess := session.Must(session.NewSession(&aws.Config{
		MaxRetries: aws.Int(3),
	}))
	svc := iam.New(sess, &aws.Config{})
	c := credentials.New(svc)

	// run list as a default
	if len(os.Args) == 1 {
		c.RunListCmd("")
		return
	}

	// run subcommands
	switch os.Args[1] {
	case "all":
		allCmd.Parse(os.Args[2:])
		c.KeyMaxAge = keyMaxAge
		c.RunAllCmd()

	case "delete":
		deleteCmd.Parse(os.Args[2:])
		keyId := deleteCmd.Arg(0)
		if keyId == "" {
			deleteCmd.Usage()
		}
		c.RunDeleteCmd(keyId, username)

	case "disable":
		disableCmd.Parse(os.Args[2:])
		keyId := disableCmd.Arg(0)
		if keyId == "" {
			disableCmd.Usage()
		}
		c.RunDisableCmd(keyId, username)

	case "enable":
		enableCmd.Parse(os.Args[2:])
		keyId := enableCmd.Arg(0)
		if keyId == "" {
			enableCmd.Usage()
		}
		c.RunEnableCmd(os.Args[2], username)

	case "help":
		usage := "Available commands:\n\n" +
			"\tall - displays a list of all keys for the current account\n" +
			"\tdelete - delete a key\n" +
			"\tdisable - deactivate a key\n" +
			"\tenable - activate a key\n" +
			"\tnew - create a new key\n" +
			"\tlist (default) - show current user keys\n\n" +
			"Use -h to get usage for a command\n"
		fmt.Printf(usage)

	case "new":
		newCmd.Parse(os.Args[2:])
		c.WriteCredentialsFile = writeFile
		c.RunNewCmd(username)

	case "list":
		listCmd.Parse(os.Args[2:])
		c.KeyMaxAge = keyMaxAge
		c.RunListCmd(username)

	default: // run list as default
		listCmd.Parse(os.Args[1:])
		c.KeyMaxAge = keyMaxAge
		c.RunListCmd(username)
	}

}
