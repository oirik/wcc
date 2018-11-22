package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"
)

const fileName = "wcc.dat"

type status string

var (
	statusNoChange status = "no change"
	statusError    status = "error"
	statusUpdated  status = "updated"
)

type wccWebsite struct {
	URL         string
	Selector    string
	Hash        string
	Status      status
	Error       error
	LastUpdated time.Time
	LastChecked time.Time
}

type wccModel struct {
	Websites []*wccWebsite
}

func open() (*wccModel, error) {
	wcc := &wccModel{}

	_, err := os.Stat(fileName)
	if err == nil {
		f, err := os.Open(fileName)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		err = gob.NewDecoder(f).Decode(wcc)
		if err != nil {
			return nil, err
		}
	}

	return wcc, nil
}

func (wcc *wccModel) fprint(output io.Writer) {
	tw := &tabwriter.Writer{}
	tw.Init(output, 0, 4, 0, ' ', 0)
	fmt.Fprint(tw, "No.\t | Status\t | URL\t | Selector\t | LastUpdated\t | LastChecked\n")
	fmt.Fprint(tw, "---\t | ------\t | ---\t | --------\t | -----------\t | -----------\n")
	for i, ws := range wcc.Websites {
		var lastUpdated, lastChecked string
		if !ws.LastUpdated.IsZero() {
			lastUpdated = ws.LastUpdated.Format("2006/01/02 15:04:05")
		}
		if !ws.LastChecked.IsZero() {
			lastChecked = ws.LastChecked.Format("2006/01/02 15:04:05")
		}
		fmt.Fprintf(tw, "%d\t | %s\t | %s\t | %s\t | %s\t | %s\n", i+1, ws.Status, ws.URL, ws.Selector, lastUpdated, lastChecked)
	}
	tw.Flush()
}

func (wcc *wccModel) save() error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	err = gob.NewEncoder(f).Encode(wcc)
	if err != nil {
		return err
	}

	return nil
}
