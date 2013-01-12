// Copyright 2012 Gonéri Le Bouder. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "notmuch"
import "log"
import "encoding/json"
import "os"
import "io"
import "fmt"
import "regexp"
import "net/mail"
import "path"

type Filter struct {
	Field   string
	Pattern string
	Re      *regexp.Regexp
	Tags    string
}

type Result struct {
	MessageID string
	Tags      string
	Die       bool
        Filename  string
}

const NCPU = 4 // number of CPU cores

func getMaildirLoc() string {
	// honor NOTMUCH_CONFIG
	home := os.Getenv("NOTMUCH_CONFIG")
	if home == "" {
		home = os.Getenv("HOME")
	}

	return path.Join(home, "Maildir")
}

func RefreshFlags(nmdb *notmuch.Database) {

	query := nmdb.CreateQuery("tag:inbox and tag:delete")
	msgs := query.SearchMessages()
	for ; msgs.Valid(); msgs.MoveToNext() {
		msg := msgs.Get()
		msg.RemoveTag("inbox")
	}

	query = nmdb.CreateQuery("tag:inbox and tag:archive")
	msgs = query.SearchMessages()
	for ; msgs.Valid(); msgs.MoveToNext() {
		msg := msgs.Get()
		msg.RemoveTag("inbox")
	}

	query = nmdb.CreateQuery("tag:inbox and tag:seen and not tag:list")
	msgs = query.SearchMessages()
	for ; msgs.Valid(); msgs.MoveToNext() {
		msg := msgs.Get()
		msg.AddTag("archive")
		msg.RemoveTag("inbox")
	}

	query = nmdb.CreateQuery("tag:inbox and tag:seen and tag:list")
	msgs = query.SearchMessages()
	for ; msgs.Valid(); msgs.MoveToNext() {
		msg := msgs.Get()
		msg.RemoveTag("inbox")
	}

	query = nmdb.CreateQuery("tag:inbox and tag:seen and tag:bug")
	msgs = query.SearchMessages()
	for ; msgs.Valid(); msgs.MoveToNext() {
		msg := msgs.Get()
		msg.RemoveTag("inbox")
	}

	query = nmdb.CreateQuery("tag:inbox and tag:killed")
	msgs = query.SearchMessages()
	for ; msgs.Valid(); msgs.MoveToNext() {
		msg := msgs.Get()
		threadId := msg.GetThreadId()
		filter := fmt.Sprintf("thread:%s", threadId)

		query := nmdb.CreateQuery(filter)
		msgs := query.SearchMessages()
		for ; msgs.Valid(); msgs.MoveToNext() {
			msg := msgs.Get()
			msg.RemoveTag("inbox")
		}
	}

	query = nmdb.CreateQuery("tag:inbox")
	msgs = query.SearchMessages()
	for ; msgs.Valid(); msgs.MoveToNext() {
		msg := msgs.Get()
		threadId := msg.GetThreadId()
		filter := fmt.Sprintf("thread:%s", threadId)

		query := nmdb.CreateQuery(filter)
		msgs := query.SearchMessages()
		for ; msgs.Valid(); msgs.MoveToNext() {
			msg := msgs.Get()
			msg.AddTag("inbox")
		}
	}

	nmdb.Close()
	fmt.Print("Ok\n")

}

func studyMsg(filter []Filter, filenameIn chan string, resultOut chan Result, quit chan bool) {
	for {
		filename := <-filenameIn

		if filename == "" {
			var result Result
			result.Die = true
			resultOut <- result

			return
		}
		// We can use Notmuch for this directly because Xappian will
		// fails as soon as we have 2 concurrent goroutine
		file, err := os.Open(filename) // For read access.
		if err != nil {
			log.Fatal(err)
		}
		var msg *mail.Message
		msg, err = mail.ReadMessage(file)

		if err != nil {
			log.Fatal(err)
		}

		var result Result
		result.MessageID = msg.Header.Get("Message-Id")
                if (result.MessageID == "") {
                    fmt.Printf("No message ID for %s\n", filename)
                    continue;
                }
		result.Filename = filename
		for _, f := range filter {
			if f.Re.MatchString(msg.Header.Get(f.Field)) {
				result.Tags += " "
				result.Tags += f.Tags
			}

		}
		file.Close()

		resultOut <- result
	}
}

func loadFilter() (filter []Filter) {

	file, err := os.Open(fmt.Sprintf("/%s/notmuch-filter.json", getMaildirLoc())) // For read access.
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	for {
		var f Filter
		if err := dec.Decode(&f); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		var err error = nil
		if f.Re, err = regexp.Compile(f.Pattern); err != nil {
			log.Printf("error: %v\n", err)
		}

		filter = append(filter, f)
	}

	return filter
}

func studyMsgs(resultOut chan Result, quit chan bool, filenames []string) {

	filter := loadFilter()

	filenameIn := make(chan string)
	for i := 0; i < NCPU+1; i++ {
		go studyMsg(filter, filenameIn, resultOut, quit)
	}
	for _, filename := range filenames {
		filenameIn <- filename
	}

	for i := 0; i < NCPU+1; i++ {
		filenameIn <- ""
	}

	quit <- true
}

func main() {
	var query *notmuch.Query
	var nmdb *notmuch.Database

	if db, status := notmuch.OpenDatabase(getMaildirLoc(),
		notmuch.DATABASE_MODE_READ_ONLY); status == notmuch.STATUS_SUCCESS {
		nmdb = db
	} else {
		log.Fatalf("Failed to open the database: %v\n", status)
	}

	quit := make(chan bool)
	resultOut := make(chan Result)

	query = nmdb.CreateQuery("tag:new")

	println(">", query.CountMessages(), "<")
	msgs := query.SearchMessages()

	var filenames []string
	if query.CountMessages() > 0 {
		for ; msgs.Valid(); msgs.MoveToNext() {
			msg := msgs.Get()

			filenames = append(filenames, msg.GetFileName())
		}
	}

	go studyMsgs(resultOut, quit, filenames)

	//	var query *notmuch.Query
	var msgIDRegexp = regexp.MustCompile("^<(.*)>$")
	var tagRegexp = regexp.MustCompile("([\\+-])(\\S+)")

	// open the database
	if db, status := notmuch.OpenDatabase(getMaildirLoc(),
		1); status == notmuch.STATUS_SUCCESS {
		nmdb = db
	} else {
		log.Fatalf("Failed to open the database: %v\n", status)
	}
	defer nmdb.Close()

	var running int = NCPU + 1
	for {
		result := <-resultOut

		if result.Die {

			running--

			if running > 0 {
				continue
			} else {
                            break
			}
		}

		// Message-ID without the <>
                fmt.Printf("MessageID: %s\n", result.MessageID)
		msgID := msgIDRegexp.FindStringSubmatch(result.MessageID)[1]
		filter := "id:"
		filter += msgID
		query := nmdb.CreateQuery(filter)
		msgs := query.SearchMessages()
		msg := msgs.Get()
		if msg == nil {
                    fmt.Printf("Can't find MessageID %s for mail %s\n", msgID, result.Filename)
                    return;
                }

		fmt.Printf("%s, tags: %s\n", msgID, result.Tags)
		msg.Freeze()
		for _, v := range tagRegexp.FindAllStringSubmatch(result.Tags, -1) {
			if v[1] == "+" {
				msg.AddTag(v[2])
			} else if v[1] == "-" {
				msg.RemoveTag(v[2])
			}
		}
		msg.Thaw()

	}
        RefreshFlags(nmdb)
        fmt.Printf("exit\n")
        os.Exit(0)


}
