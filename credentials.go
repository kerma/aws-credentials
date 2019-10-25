package credentials

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/fatih/color"
)

type AccessKeys struct {
	// *iam.AccessKeyMetadata
	UserName    string
	AccessKeyId string
	Status      string
	CreateDate  time.Time

	Loaded   bool
	KeyAge   int
	IsOld    bool
	LastUsed *time.Time
}

type Config struct {
	svc                  *iam.IAM
	KeyMaxAge            int
	WriteCredentialsFile bool
}

func New(s *iam.IAM) *Config {
	return &Config{
		svc: s,
	}
}

var (
	red   = color.New(color.FgRed)
	white = color.New(color.FgWhite)
	green = color.New(color.FgGreen)
)

// RunListCmd runs default command to check user access keys
func (c *Config) RunListCmd(username string) {
	keys := c.getAccessKeys(username)
	fmt.Println()
	printAccessKeys(keys)
}

// RunAllCmd retrieves all users and their access keys
func (c *Config) RunAllCmd() {
	keys := []AccessKeys{}
	fmt.Println("Retrieving keys for all users, please wait...")
	for _, username := range c.getAllUsernames() {
		for _, k := range c.getAccessKeys(username) {
			keys = append(keys, k)
		}
	}
	fmt.Println()
	printAccessKeys(keys)
}

// RunNewCmd creates a new access key and writes it to credentials file
func (c *Config) RunNewCmd(username string) {
	in := &iam.CreateAccessKeyInput{}
	if username != "" {
		in.UserName = &username
	}
	fmt.Println("Generating new access key")
	out, err := c.svc.CreateAccessKey(in)
	if err != nil {
		fatal(err)
	}
	fmt.Print("Access key generated: ")
	color.Green(*out.AccessKey.AccessKeyId)

	credsOutput := formatCredentialsFile(out.AccessKey)
	if c.WriteCredentialsFile {
		fmt.Println("Writing credentials file")
		err = writeCredentialsFile(credsOutput)
		if err != nil {
			fatal(err)
		}
		fmt.Println("~/.aws/credentials updated: ")
		fmt.Println(credsOutput)

		fmt.Println("Activate new profile with: ")
		fmt.Printf("\texport AWS_PROFILE=%v.new\n\n", getProfile())
	} else {
		fmt.Printf("Activate with:\n\n")
		fmt.Println(formatConsoleExport(out.AccessKey))
	}

	creds, _ := c.svc.Config.Credentials.Get()
	fmt.Println("Delete old key with:")
	fmt.Printf("\taws-credentials delete -k %v\n", creds.AccessKeyID)

}

// RunDeleteCmd deletes a given access key
func (c *Config) RunDeleteCmd(key string) {
	_, err := c.svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{AccessKeyId: &key})
	if err != nil {
		fatal(err)
	}
	red.Printf("%s deleted.\n", key)
}

// RunDisableCmd deactivates a given access key
func (c *Config) RunDisableCmd(key string) {
	status := "Inactive"
	_, err := c.svc.UpdateAccessKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: &key,
		Status:      &status,
	})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("%s deactivated.\n", key)
}

// RunEnableCmd activates a given access key
func (c *Config) RunEnableCmd(key string) {
	status := "Active"
	_, err := c.svc.UpdateAccessKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: &key,
		Status:      &status,
	})
	if err != nil {
		fatal(err)
	}
	green.Printf("%s activated.\n", key)
}

func (c *Config) getAllUsernames() []string {
	usernames := []string{}
	in := &iam.ListUsersInput{}
	for {
		resp, err := c.svc.ListUsers(in)
		if err != nil {
			fatal(err)
		}
		for _, u := range resp.Users {
			usernames = append(usernames, *u.UserName)
		}
		if *resp.IsTruncated {
			in.SetMarker(*resp.Marker)
			continue
		}
		break
	}
	return usernames
}

func (c *Config) getAccessKeys(user string) []AccessKeys {
	in := &iam.ListAccessKeysInput{}
	if user != "" {
		in.UserName = &user
	}
	out, err := c.svc.ListAccessKeys(in)
	if err != nil {
		fatal(err)
	}

	creds, _ := c.svc.Config.Credentials.Get()
	keys := []AccessKeys{}
	for _, meta := range out.AccessKeyMetadata {
		ak := AccessKeys{
			UserName:    *meta.UserName,
			AccessKeyId: *meta.AccessKeyId,
			Status:      *meta.Status,
			CreateDate:  *meta.CreateDate,
			Loaded:      *meta.AccessKeyId == creds.AccessKeyID,
			KeyAge:      daysSince(*meta.CreateDate),
			IsOld:       olderThan(*meta.CreateDate, c.KeyMaxAge),
		}
		resp, err := c.svc.GetAccessKeyLastUsed(
			&iam.GetAccessKeyLastUsedInput{AccessKeyId: meta.AccessKeyId})
		if err != nil {
			fmt.Println("Couldn't get usage data for ", meta.AccessKeyId)
		} else {
			ak.LastUsed = resp.AccessKeyLastUsed.LastUsedDate
		}
		keys = append(keys, ak)
	}
	return keys
}

func daysSince(t time.Time) int {
	return int(time.Since(t).Hours() / 24)
}

func olderThan(t time.Time, d int) bool {
	return daysSince(t) > d
}

func printAccessKeys(keys []AccessKeys) {

	white.Printf("   %-20s\t%-4s\t%-8s\t%-30s\t%s\n",
		"AccessKeyId", "Age", "Status", "LastUsed", "Username")
	for _, k := range keys {
		if k.Loaded {
			green.Printf("✔  ")
		} else {
			fmt.Printf("   ")
		}

		fmt.Printf("%-20s\t", k.AccessKeyId)

		if k.IsOld {
			red.Printf("%-4d\t", k.KeyAge)
		} else {
			fmt.Printf("%-4d\t", k.KeyAge)
		}

		if k.Status == "Active" {
			green.Printf("%-8s\t", k.Status)
		} else {
			fmt.Printf("%-8s\t", k.Status)
		}

		if k.LastUsed != nil {
			fmt.Printf("%-30v\t", k.LastUsed)
		} else {
			fmt.Printf("%-30s\t", "never")
		}
		fmt.Println(k.UserName)
	}
}

func getProfile() string {
	p := os.Getenv("AWS_PROFILE")
	if p != "" {
		return p
	}
	return "default"
}

func formatCredentialsFile(k *iam.AccessKey) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "\n[%v.new]\n", getProfile())
	fmt.Fprintf(b, "aws_access_key_id=%v\n", *k.AccessKeyId)
	fmt.Fprintf(b, "aws_secret_access_key=%v\n", *k.SecretAccessKey)
	return b.String()
}

func formatConsoleExport(k *iam.AccessKey) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "\texport AWS_ACCESS_KEY_ID=%v\n", *k.AccessKeyId)
	fmt.Fprintf(b, "\texport AWS_SECRET_ACCESS_KEY=%v\n", *k.SecretAccessKey)
	return b.String()
}

func writeCredentialsFile(s string) error {
	path, _ := os.UserHomeDir()
	path = path + "/.aws/credentials"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(s); err != nil {
		return err
	}

	return nil
}

func fatal(i ...interface{}) {
	fmt.Println(i...)
	os.Exit(1)
}
