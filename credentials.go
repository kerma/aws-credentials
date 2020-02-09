package credentials

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/fatih/color"
)

const (
	DefaultKeyMaxAge = 90
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
		svc:       s,
		KeyMaxAge: DefaultKeyMaxAge,
	}
}

var (
	red   = color.New(color.FgRed)
	white = color.New(color.FgWhite)
	green = color.New(color.FgGreen)
)

// RunListCmd runs default command to check current user access keys
func (c *Config) RunListCmd() int {
	keys, err := c.getCurrentAccessKeys()
	if err != nil {
		return fatal(err)
	}
	fmt.Println()
	printAccessKeys(keys)
	return 0
}

// RunUserListCmd runs command to check given user access keys
func (c *Config) RunUserListCmd(username string) int {
	keys, err := c.getUserAccessKeys(username)
	if err != nil {
		return fatal(err)
	}
	fmt.Println()
	printAccessKeys(keys)
	return 0
}

// RunAllCmd retrieves all users and their access keys
func (c *Config) RunAllCmd() int {
	var keys []AccessKeys
	fmt.Println("Retrieving keys for all users, please wait...")
	for _, username := range c.getAllUsernames() {
		uk, err := c.getUserAccessKeys(username)
		if err != nil {
			return fatal(err)
		}
		for _, k := range uk {
			keys = append(keys, k)
		}
	}
	fmt.Println()
	printAccessKeys(keys)
	return 0
}

func (c *Config) RunCheckKeys() int {
	keys, err := c.getCurrentAccessKeys()
	if err != nil {
		return fatal(err)
	}
	return checkAccessKeys(keys)
}

func (c *Config) RunCheckAllKeys() int {
	var keys []AccessKeys
	for _, username := range c.getAllUsernames() {
		uk, err := c.getUserAccessKeys(username)
		if err != nil {
			return fatal(err)
		}
		for _, k := range uk {
			keys = append(keys, k)
		}
	}
	return checkAccessKeys(keys)
}

// RunNewCmd creates a new access key
func (c *Config) RunNewCmd() int {
	err := c.createNewKey(&iam.CreateAccessKeyInput{})
	if err != nil {
		return fatal(err)
	}
	return 0
}

// RunUserNewCmd creates a new access key for the given user
func (c *Config) RunUserNewCmd(username string) int {
	in := &iam.CreateAccessKeyInput{
		UserName: &username,
	}
	err := c.createNewKey(in)
	if err != nil {
		return fatal(err)
	}
	return 0
}

func (c *Config) createNewKey(in *iam.CreateAccessKeyInput) error {
	fmt.Println("Generating new access key")
	out, err := c.svc.CreateAccessKey(in)
	if err != nil {
		return err
	}
	fmt.Print("Access key generated: ")
	color.Green(*out.AccessKey.AccessKeyId)

	credsOutput := formatCredentialsFile(out.AccessKey)
	if c.WriteCredentialsFile {
		fmt.Println("Writing credentials file")
		err = writeCredentialsFile(credsOutput)
		if err != nil {
			return err
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
	fmt.Printf("\tcredentials delete %v\n", creds.AccessKeyID)

	return nil
}

// RunDeleteUserKeyCmd deletes a given users access key
func (c *Config) RunDeleteUserKeyCmd(key, username string) int {
	return c.deleteKey(&iam.DeleteAccessKeyInput{
		AccessKeyId: &key,
		UserName:    &username,
	})
}

// RunDeleteCmd deletes a given access key
func (c *Config) RunDeleteCmd(key string) int {
	return c.deleteKey(&iam.DeleteAccessKeyInput{
		AccessKeyId: &key,
	})
}

func (c *Config) deleteKey(in *iam.DeleteAccessKeyInput) int {
	_, err := c.svc.DeleteAccessKey(in)
	if err != nil {
		return fatal(err)
	}
	red.Printf("%s deleted.\n", in.AccessKeyId)
	return 0
}

// RunDisableCmd deactivates a given access key
func (c *Config) RunDisableCmd(key string) int {
	status := "Inactive"
	return c.disableKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: &key,
		Status:      &status,
	})
}

// RunDisableUserKeyCmd deactivates a given user access key
func (c *Config) RunDisableUserKeyCmd(key, username string) int {
	status := "Inactive"
	return c.disableKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: &key,
		Status:      &status,
		UserName:    &username,
	})
}

func (c *Config) disableKey(in *iam.UpdateAccessKeyInput) int {
	_, err := c.svc.UpdateAccessKey(in)
	if err != nil {
		return fatal(err)
	}
	fmt.Printf("%s deactivated.\n", in.AccessKeyId)
	return 0
}

// RunEnableCmd activates a given access key
func (c *Config) RunEnableCmd(key string) int {
	status := "Active"
	return c.enableKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: &key,
		Status:      &status,
	})
}

// RunEnableCmd activates a given access key
func (c *Config) RunEnableUserKeyCmd(key, username string) int {
	status := "Active"
	return c.enableKey(&iam.UpdateAccessKeyInput{
		AccessKeyId: &key,
		Status:      &status,
		UserName:    &username,
	})
}

func (c *Config) enableKey(in *iam.UpdateAccessKeyInput) int {
	_, err := c.svc.UpdateAccessKey(in)
	if err != nil {
		return fatal(err)
	}
	green.Printf("%s activated.\n", in.AccessKeyId)
	return 0
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

func (c *Config) getCurrentAccessKeys() ([]AccessKeys, error) {
	return c.getAccessKeys(&iam.ListAccessKeysInput{})
}

func (c *Config) getUserAccessKeys(user string) ([]AccessKeys, error) {
	in := &iam.ListAccessKeysInput{
		UserName: &user,
	}
	return c.getAccessKeys(in)
}

func (c *Config) getAccessKeys(in *iam.ListAccessKeysInput) ([]AccessKeys, error) {
	out, err := c.svc.ListAccessKeys(in)
	if err != nil {
		return nil, err
	}

	creds, _ := c.svc.Config.Credentials.Get()
	var keys []AccessKeys
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
		resp, ierr := c.svc.GetAccessKeyLastUsed(
			&iam.GetAccessKeyLastUsedInput{AccessKeyId: meta.AccessKeyId})
		if ierr != nil {
			fmt.Println("Couldn't get usage data for ", meta.AccessKeyId)
			err = ierr
		} else {
			ak.LastUsed = resp.AccessKeyLastUsed.LastUsedDate
		}
		keys = append(keys, ak)
	}
	return keys, err
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

func checkAccessKeys(keys []AccessKeys) int {
	var b bytes.Buffer
	ret := 0
	w := bufio.NewWriter(&b)
	fmt.Fprintf(w, "%-20s\t%-4s\t%-8s\t%-30s\t%s\n",
		"AccessKeyId", "Age", "Status", "LastUsed", "Username")
	for _, k := range keys {
		if k.IsOld {
			ret = 1
			fmt.Fprintf(w, "%-20s\t", k.AccessKeyId)
			fmt.Fprintf(w, "%-4d\t", k.KeyAge)
			fmt.Fprintf(w, "%-8s\t", k.Status)
			if k.LastUsed != nil {
				fmt.Fprintf(w, "%-30v\t", k.LastUsed)
			} else {
				fmt.Fprintf(w, "%-30s\t", "never")
			}
			fmt.Fprintln(w, k.UserName)
		}
	}
	w.Flush()
	if ret != 0 {
		fmt.Print(b.String())
	}
	return ret
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

func fatal(i ...interface{}) int {
	fmt.Println(i...)
	return 1
}
