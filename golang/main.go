package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/austinorth/flag"
	"github.com/google/go-github/v28/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/k0kubun/pp"
	"github.com/nozzle/throttler"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

//  go run github.go -token=02406a1e4eeab7ee947329f2c9b6ed6c9a81549d -query=arxiv

func generateDateRange(query string, startYear, endYear int) (queries []string) {
	// for curYear := startYear; curYear >= startYear; curYear++ {
	for curYear := endYear; curYear >= startYear; curYear-- {
		log.Println("curYear", curYear, "endYear", endYear, "startYear", startYear)
		dateRange := fmt.Sprintf("created:%d-01-01..%d-12-31", curYear, curYear)
		queries = append(queries, fmt.Sprintf("%s %s", query, dateRange))
	}
	return
}

func main() {
	token := flag.String("token", "", "Personal/Oauth2 github token")
	query := flag.String("query", "", "Query")
	flag.Parse()

	// Setup Github API client, with persistent caching
	var (
		cache          = diskcache.New("./data/github-cache")
		cacheTransport = httpcache.NewTransport(cache)
		tokenSource    = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *token})
		authTransport  = oauth2.Transport{Source: tokenSource, Base: cacheTransport}
		client         = github.NewClient(&http.Client{Transport: &authTransport})
	)

	ctx := context.Background()
	// tc := oauth2.NewClient(ctx, ts)
	// client := github.NewClient(tc)

	searchOpt := &github.SearchOptions{
		Sort:  "created",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	reposSet := make(map[string]struct{})

	queries := generateDateRange(*query, 2007, 2020)
	pp.Println(queries)
	// os.Exit(1)

	t := throttler.New(1, 10000000)

	for _, q := range queries {
		fmt.Println("query:", q)
		currentPage := 1
		lastPage := 1

		for currentPage <= lastPage {
			// Increment the WaitGroup counter.
			// wg.Add(1)
			go func(q string) error {
				// Let Throttler know when the goroutine completes
				// so it can dispatch another worker
				defer t.Done(nil)
				// time.Sleep(4 * time.Second)
				// code, resp, err := client.Search.Code(ctx, query, searchOpt)
				code, resp, err := client.Search.Repositories(ctx, q, searchOpt)
				sleepIfRateLimitExceeded(ctx, client)
				if err != nil {
					return err
				}

				lastPage = resp.LastPage
				log.Println("visiting ", resp.Request.URL.String())
				// log.Println("lastPage: ", resp.LastPage, "nextPage: ", resp.NextPage, "currentPage", currentPage, "lastPage", lastPage)
				// for _, cr := range code.CodeResults {
				for _, cr := range code.Repositories {
					repoURL := *cr.HTMLURL
					if _, ok := reposSet[repoURL]; !ok {
						reposSet[repoURL] = struct{}{}
					}
				}
				currentPage++
				if resp.LastPage == 0 {
					return nil
				}
				searchOpt.Page = resp.NextPage
				// Go to the next page
				return nil
			}(q)
			// Wait for all HTTP fetches to complete.
			t.Throttle()
		}
		searchOpt.Page = 1
		currentPage = 1
		lastPage = 1
	}

	if t.Err() != nil {
		// Loop through the errors to see the details
		for i, err := range t.Errs() {
			fmt.Printf("error #%d: %s", i, err)
		}
		log.Fatal(t.Err())
	}
count := false
	if count {
	c := 0
	for r := range reposSet {
		fmt.Println(r)
		c++
	}
	log.Println("count: ", c)
}
}

func sleepIfRateLimitExceeded(ctx context.Context, client *github.Client) {
	rateLimit, _, err := client.RateLimits(ctx)
	if err != nil {
		fmt.Printf("Problem in getting rate limit information %v\n", err)
		return
	}

	if rateLimit.Search.Remaining == 1 {
		timeToSleep := rateLimit.Search.Reset.Sub(time.Now()) + time.Second
		time.Sleep(timeToSleep)
	}
}

type outputFunc func(fileName, line string, match []string)

func writeOutput(output *csv.Writer, repoURL string) outputFunc {
	return func(fileName, line string, match []string) {
		record := []string{repoURL, fileName}
		if len(match) == 0 {
			record = append(record, line)
		} else {
			record = append(record, match[1:]...)
		}
		output.Write(record)
		output.Flush()
	}
}

func findInRepo(log logrus.FieldLogger, output outputFunc, repoURL string, auth transport.AuthMethod, pattern *regexp.Regexp) error {
	fs := memfs.New()
	storer := memory.NewStorage()

	repo, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:   repoURL,
		Auth:  auth,
		Depth: 1,
	})
	if err != nil {
		return fmt.Errorf("can not clone: %s", err)
	}

	tree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("can not get worktree: %s", err)
	}

	dir, err := fs.ReadDir("")
	if err != nil {
		return fmt.Errorf("can not read root: %s", err)
	}

	return walkInfo(log, output, tree.Filesystem, "", dir, pattern)
}

func walkInfo(log logrus.FieldLogger, output outputFunc, fs billy.Filesystem, base string, dir []os.FileInfo, pattern *regexp.Regexp) error {
	for _, info := range dir {
		name := filepath.Join(base, info.Name())
		if info.IsDir() {
			if err := findInDir(log.WithField("dir", name), output, fs, name, pattern); err != nil {
				log.Warnf("error finding in %q: %s", name, err)
			}
			continue
		}

		if err := findInFile(log.WithField("file", name), output, fs, name, pattern); err != nil {
			log.Warnf("error finding in %q: %s", name, err)
		}
	}

	return nil
}

func findInDir(log logrus.FieldLogger, output outputFunc, fs billy.Filesystem, dir string, pattern *regexp.Regexp) error {
	log.Debugf("dir: %s", dir)

	subDir, err := fs.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("can not list %q: %s", dir, err)
	}

	return walkInfo(log, output, fs, dir, subDir, pattern)
}

func findInFile(log logrus.FieldLogger, output outputFunc, fs billy.Filesystem, name string, pattern *regexp.Regexp) error {
	log.Debugf("file: %s", name)
	file, err := fs.Open(name)
	if err != nil {
		return fmt.Errorf("can not read %q: %s", name, err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if match := pattern.FindStringSubmatch(line); match != nil {
			output(name, line, match)
		}
	}
	return nil
}
