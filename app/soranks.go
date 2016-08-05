package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/klashxx/soranks/lib"
)

const (
	MaxErrors     = 3
	MaxPages      = 1100
	MinReputation = 500
	APIKeyPath    = "../_secret/api.key"
	GitHubToken   = "../_secret/token"
	SOApiURL      = "https://api.stackexchange.com/2.2"
	SOUsersQuery  = `users?page=%d&pagesize=100&order=desc&sort=reputation&site=stackoverflow`
	SOUserTags    = `users/%d/top-answer-tags?page=1&pagesize=3&site=stackoverflow`
	GHApiURL      = "https://api.github.com/repos/klashxx/soranks"
)

var (
	author   = lib.Committer{Name: "klasxx", Email: "klashxx@gmail.com"}
	branch   = "dev"
	location = flag.String("location", ".", "location")
	jsonfile = flag.String("json", "", "json sample file")
	jsonrsp  = flag.String("jsonrsp", "", "json response file")
	mdrsp    = flag.String("mdrsp", "", "markdown response file")
	limit    = flag.Int("limit", 20, "max number of records")
	term     = flag.Bool("term", false, "print output in terminal")
	publish  = flag.String("publish", "", "publish ranks in Github")
)

func main() {
	flag.Parse()
	lib.Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)

	lib.Trace.Println("location: ", *location)
	lib.Trace.Println("json: ", *jsonfile)
	lib.Trace.Println("jsontest: ", *jsonfile)
	lib.Trace.Println("jsonrsp: ", *jsonrsp)
	lib.Trace.Println("mdrsp: ", *mdrsp)
	lib.Trace.Println("limit: ", *limit)
	lib.Trace.Println("term: ", *term)
	lib.Trace.Println("publish: ", *publish)

	if *publish != "" && *mdrsp == "" {
		lib.Error.Println("Publish requires mdrsp!!")
		os.Exit(5)
	}

	re := regexp.MustCompile(fmt.Sprintf("(?i)%s", *location))

	stop := false
	streamErrors := 0
	currentPage := 1
	lastPage := currentPage
	counter := 0

	var users *lib.SOUsers
	var ranks lib.Ranks
	var key string
	var err error

	for {
		if *jsonfile == "" {
			if lastPage == currentPage {
				lib.Info.Println("Trying to extract API key.")
				key = fmt.Sprintf("&key=%s", lib.GetKey(APIKeyPath))
			}

			lib.Trace.Printf("Requesting page: %d\n", currentPage)

			url := fmt.Sprintf("%s/%s%s", SOApiURL, fmt.Sprintf(SOUsersQuery, currentPage), key)

			users = new(lib.SOUsers)

			err = lib.StreamHTTP(url, users, true)

			lib.Trace.Printf("Page users: %d\n", len(users.Items))
			if err != nil || len(users.Items) == 0 {

				lib.Warning.Println("Can't stream data.")
				streamErrors += 1
				if streamErrors >= MaxErrors {
					lib.Error.Println("Max retry number reached")
					os.Exit(5)
				}
				continue
			}
		} else {
			lib.Info.Println("Extracting from source JSON file.")
			var err error
			users, err = lib.StreamFile(*jsonfile)
			if err != nil {
				lib.Error.Println("Can't decode json file.")
				os.Exit(5)
			}
			stop = true
		}

		lib.Trace.Println("User info extraction.")

		repLimit := lib.GetUserInfo(users, MinReputation, re, &counter, *limit, &ranks, *term)
		if !repLimit {
			break
		}
		lib.Trace.Println("User info extraction done.")

		lastPage = currentPage
		currentPage += 1
		if (currentPage >= MaxPages && MaxPages != 0) || !users.HasMore || stop {
			break
		}
	}

	if counter == 0 {
		lib.Warning.Println("No results found.")
		os.Exit(0)
	}

	if *jsonrsp != "" {
		lib.DumpJson(jsonrsp, &ranks)
	}

	if *mdrsp != "" {
		lib.DumpMarkdown(mdrsp, ranks, location)
		if *publish != "" {
			_ = lib.GitHubConnector(GHApiURL, *publish, *mdrsp, GitHubToken, branch, author)
		}
	}

	lib.Info.Printf("%04d pages requested.\n", lastPage)
	lib.Info.Printf("%04d users found.\n", counter)
}