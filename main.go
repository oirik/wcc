package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"crypto/md5"

	"github.com/PuerkitoBio/goquery"
	"github.com/oirik/gosubcommand"
)

var (
	version  string
	revision string
)

const maxCheckCount = 5

func main() {
	gosubcommand.AppName = "wcc"
	gosubcommand.Version = fmt.Sprintf("version: %s\nrevision: %s", version, revision)
	gosubcommand.Summary = "wcc is a command-line tool which checks whether a website is changed or not. \nThis tool creates a data file named `wcc.dat` to save the registered websites in a current direcotry."

	gosubcommand.Register("add", &addCommand{})
	gosubcommand.Register("rm", &rmCommand{})
	gosubcommand.Register("list", &listCommand{})
	gosubcommand.Register("check", &checkCommand{})

	os.Exit(int(gosubcommand.Execute()))
}

type addCommand struct{}

func (cmd *addCommand) Summary() string          { return "Add new website which you want to check updates." }
func (cmd *addCommand) SetFlag(fs *flag.FlagSet) {}
func (cmd *addCommand) Execute(fs *flag.FlagSet) gosubcommand.ExitCode {
	url := fs.Arg(0)
	if url == "" {
		fmt.Fprintln(os.Stderr, "url argument is missing.")
		return gosubcommand.ExitCodeUsageError
	}
	selector := fs.Arg(1)

	wcc, err := open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open data file.", err)
		return gosubcommand.ExitCodeError
	}

	hash, err := getHash(url, selector)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to access with the url+selector", err)
		return gosubcommand.ExitCodeError
	}

	wcc.Websites = append(wcc.Websites, &wccWebsite{URL: url, Selector: selector, Hash: hash})
	err = wcc.save()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to save data file.", err)
		return gosubcommand.ExitCodeError
	}

	wcc.fprint(os.Stdout)

	return gosubcommand.ExitCodeSuccess
}

type rmCommand struct{}

func (cmd *rmCommand) Summary() string          { return "Remove website from the registered list." }
func (cmd *rmCommand) SetFlag(fs *flag.FlagSet) {}
func (cmd *rmCommand) Execute(fs *flag.FlagSet) gosubcommand.ExitCode {
	index, err := strconv.ParseInt(fs.Arg(0), 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, "argument is not integer.", err)
		return gosubcommand.ExitCodeUsageError
	}

	wcc, err := open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open data file.", err)
		return gosubcommand.ExitCodeError
	}

	if index <= 0 || int(index) > len(wcc.Websites) {
		fmt.Fprintln(os.Stderr, "index is out of range.")
		return gosubcommand.ExitCodeUsageError
	}

	wcc.Websites = append(wcc.Websites[:index-1], wcc.Websites[index:]...)

	err = wcc.save()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to save data file.", err)
		return gosubcommand.ExitCodeError
	}

	wcc.fprint(os.Stdout)

	return gosubcommand.ExitCodeSuccess
}

type listCommand struct{}

func (cmd *listCommand) Summary() string          { return "Show list of the registered websites." }
func (cmd *listCommand) SetFlag(fs *flag.FlagSet) {}
func (cmd *listCommand) Execute(fs *flag.FlagSet) gosubcommand.ExitCode {
	wcc, err := open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open data file.", err)
		return gosubcommand.ExitCodeError
	}

	wcc.fprint(os.Stdout)

	return gosubcommand.ExitCodeSuccess
}

type checkCommand struct {
	slack string
}

func (cmd *checkCommand) Summary() string { return "Check updates of the registered websites." }
func (cmd *checkCommand) SetFlag(fs *flag.FlagSet) {
	fs.StringVar(&cmd.slack, "slack", "", "Set the slack's incoming webhook URL if you want to be notified when found updated")
}
func (cmd *checkCommand) Execute(fs *flag.FlagSet) gosubcommand.ExitCode {
	wcc, err := open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open data file.", err)
		return gosubcommand.ExitCodeError
	}
	if len(wcc.Websites) == 0 {
		return gosubcommand.ExitCodeSuccess
	}

	limit := make(chan struct{}, maxCheckCount)

	var wg sync.WaitGroup
	for i := range wcc.Websites {
		wg.Add(1)
		go func(i int) {
			limit <- struct{}{}
			defer wg.Done()

			now := time.Now()
			ws := wcc.Websites[i]

			ws.LastChecked = now
			hash, err := getHash(ws.URL, ws.Selector)
			if err != nil {
				ws.Status = statusError
				ws.Error = err
				return
			}

			ws.Error = nil

			if hash == ws.Hash {
				ws.Status = statusNoChange
				return
			}

			ws.Status = statusUpdated
			ws.LastUpdated = now
			ws.Hash = hash

			<-limit
		}(i)
	}
	wg.Wait()

	err = wcc.save()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to save data file.", err)
		return gosubcommand.ExitCodeError
	}

	var errorCount, updatedCount int
	for _, ws := range wcc.Websites {
		switch ws.Status {
		case statusError:
			errorCount++
		case statusUpdated:
			updatedCount++
		}
	}
	if errorCount > 0 || updatedCount > 0 {
		builder := new(strings.Builder)
		if errorCount > 0 {
			fmt.Fprintf(builder, "%d websites got error.\n\n", errorCount)
		}
		if updatedCount > 0 {
			fmt.Fprintf(builder, "%d websites have been updated.\n\n", updatedCount)
		}
		fmt.Fprintln(builder, "```")
		wcc.fprint(builder)
		fmt.Fprintln(builder, "```")

		message := builder.String()
		fmt.Println(message)

		if cmd.slack != "" {
			if err = notifySlack(message, cmd.slack); err != nil {
				panic(err)
			}
		}
	}

	return gosubcommand.ExitCodeSuccess
}

func notifySlack(message string, url string) error {
	jsonBytes, _ := json.Marshal(map[string]string{
		"text": message,
	})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return errors.Wrap(err, "failed to notify slack")
	}
	defer resp.Body.Close()

	return nil
}

func getHash(url string, selector string) (hash string, err error) {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return "", errors.Wrap(err, "failed to get the website")
	}
	var s *goquery.Selection
	if selector == "" {
		s = doc.First()
	} else {
		s = doc.Find(selector).First()
	}
	if s == nil {
		return "", errors.New("this selector don't find anything")
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(s.Text()))), nil
}
