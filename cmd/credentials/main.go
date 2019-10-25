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

const (
	defaultKeyMaxAge = 90
)

var (
	keyMaxAge              int
	credentialReportMaxAge int
	allKeys                bool
	username               string

	keyId     string
	writeFile bool
)

func main() {

	// command line flags setup
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listCmd.IntVar(&keyMaxAge, "max-age", defaultKeyMaxAge, "Maximum access key age in days")
	listCmd.IntVar(&keyMaxAge, "m", defaultKeyMaxAge, "Maximum access key age in days (shorthand)")
	listCmd.StringVar(&username, "u", "", "Username for the query (shorthand")
	listCmd.StringVar(&username, "username", "", "Username for the query")

	allCmd := flag.NewFlagSet("all", flag.ExitOnError)
	allCmd.IntVar(&keyMaxAge, "max-age", defaultKeyMaxAge, "Maximum access key age in days")
	allCmd.IntVar(&keyMaxAge, "m", defaultKeyMaxAge, "Maximum access key age in days (shorthand)")

	newCmd := flag.NewFlagSet("rotate", flag.ExitOnError)
	newCmd.StringVar(&keyId, "id", "", "Access Key ID")
	newCmd.BoolVar(&writeFile, "w", false, "Write new key to credentials file")

	// commands without flags usage setup
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  credentials delete <access-key-id>\n")
	}

	disableCmd := flag.NewFlagSet("disable", flag.ExitOnError)
	disableCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  credentials disable <access-key-id>\n")
	}

	enableCmd := flag.NewFlagSet("enable", flag.ExitOnError)
	enableCmd.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n  credentials enable <access-key-id>\n")
	}

	// session setup
	sess := session.Must(session.NewSession(&aws.Config{
		MaxRetries: aws.Int(3),
	}))
	svc := iam.New(sess, &aws.Config{})
	c := credentials.New(svc)

	// run list as a default
	if len(os.Args) == 1 {
		listCmd.Parse(os.Args[1:]) // still call parse to populate defaults
		c.KeyMaxAge = keyMaxAge
		c.RunListCmd(username)
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
		if len(os.Args) != 3 {
			deleteCmd.Usage()
		} else {
			c.RunDeleteCmd(os.Args[2])
		}

	case "disable":
		disableCmd.Parse(os.Args[2:])
		if len(os.Args) != 3 {
			disableCmd.Usage()
		} else {
			c.RunDisableCmd(os.Args[2])
		}

	case "enable":
		enableCmd.Parse(os.Args[2:])
		if len(os.Args) != 3 {
			enableCmd.Usage()
		} else {
			c.RunEnableCmd(os.Args[2])
		}

	case "new":
		newCmd.Parse(os.Args[2:])
		c.WriteCredentialsFile = writeFile
		c.RunNewCmd(username)
	case "list":
		listCmd.Parse(os.Args[1:])
		c.KeyMaxAge = keyMaxAge
		c.RunListCmd(username)

	default:
		usage := "Available commands:\n\n" +
			"\tall - displays a list of all keys for the current account\n" +
			"\tdelete - delete a key\n" +
			"\tdisable - deactivate a key\n" +
			"\tenable - activate a key\n" +
			"\tnew - create a new key\n" +
			"\tlist (default) - show current user keys\n\n" +
			"Use -h to get usage for a command\n"
		fmt.Printf(usage)
	}

}
