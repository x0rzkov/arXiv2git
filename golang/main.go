package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/k0kubun/pp"
	"github.com/nozzle/throttler"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"github.com/spf13/pflag"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"github.com/x0rzkov/go-vcsurl"
	ghttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

var (
	logLevelStr  string
	token         string
	query string
	pattern string
	debug  bool
	help bool
	httpTimeout  time.Duration
	log *logrus.Logger
)

func init() {
		log = &logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.TextFormatter{
			DisableTimestamp: true,
		},
	}

}

func main() {
	pflag.StringVarP(&logLevelStr, "log-level", "v", "info", "Logging level.")
	pflag.StringVarP(&token, "token", "t", "", "Github personal token")
	pflag.StringVarP(&query, "query", "q", "", "query")
	pflag.StringVarP(&pattern, "pattern", "p", "", "pattern (eg. Dockerfile)")
	pflag.BoolVarP(&debug, "debug", "d", false, "debug mode")
	pflag.BoolVarP(&help, "help", "h", false, "help info")
	pflag.DurationVar(&httpTimeout, "http-timeout", 5*time.Second, "Timeout for HTTP Requests.")
	if help {
		pflag.PrintDefaults()
		os.Exit(1)
	}
	pflag.Parse()


	logLevel, err := logrus.ParseLevel(logLevelStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can not parse log level %q: %s", logLevelStr, err)
		os.Exit(1)
	}

	log.Level = logLevel

	// args := pflag.Args()
	// pp.Println(args)
	// if len(args) == 0 {
	//	log.Fatal("no patterns passed")
	// }

	// patternStr := strings.Join(args, " ")
	patternStr := "Dockerfile"
	pattern, err := regexp.Compile(patternStr)
	if err != nil {
		log.Fatalf("Can not parse %q: %s", patternStr, err)
	}
	log.Infof("Pattern: %s", pattern)	

	header := []string{"repo", "file"}
	if pattern.NumSubexp() == 0 {
		header = append(header, "line")
	} else {
		for i := 0; i < pattern.NumSubexp(); i++ {
			header = append(header, fmt.Sprintf("group%d", i))
		}
	}

	output := csv.NewWriter(os.Stdout)
	output.Write(header)
	output.Flush()

	pp.Println("token:",token)

	// Setup Github API client, with persistent caching
	var (
		cache          = diskcache.New("./data/github-cache2")
		cacheTransport = httpcache.NewTransport(cache)
		tokenSource    = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		authTransport  = oauth2.Transport{Source: tokenSource, Base: cacheTransport}
		client         = github.NewClient(&http.Client{Transport: &authTransport})
	)

	ctx := context.Background()

	searchOpt := &github.SearchOptions{
		Sort:  "created",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	reposSet := make(map[string]struct{})

	queries := generateDateRange(query, 2020, 2020)
	if debug {
	pp.Println(queries)
}

	t := throttler.New(1, 10000000)

	for _, q := range queries {
		log.Println("query:", q)
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
				// sleepIfRateLimitExceeded(ctx, client)
				waitForRemainingLimit(client, false, 10)
				if err != nil {
					return err
				}

				lastPage = resp.LastPage
				log.Println("visiting", resp.Request.URL.String())
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

	repoAuth := &ghttp.BasicAuth{
		Username: "x0rzkov",
		Password: token,
	}

	patterns := []string{"Dockerfile", "docker-compose"}

	c := 0
	for repoURL := range reposSet {
		fmt.Println("repoURL", repoURL)
		log.Infof("Searching: %s", repoURL)
		if info, err := vcsurl.Parse(repoURL); err == nil {
			go func(username, name string) error {
				// Let Throttler know when the goroutine completes
				// so it can dispatch another worker
				defer t.Done(nil)
				branches, err := listBranches(client, username, name)
				if err != nil {
					log.Fatal(err)
					return err
				}
				for _, branch := range branches {
					entries, err := getEntries(client, info.Username, info.Name, branch, true)
					if err != nil {
						log.Fatal(err)
						return err
					}
					// pp.Println(entries)
					matches := matchPatterns(entries, patterns...)
					if len(matches) > 0 {
						pp.Println(matches)
					}
				}
				return nil
			}(info.Username, info.Name)
			t.Throttle()
		}
		cloneRepo := false
		if cloneRepo {
			if err := findInRepo(log.WithField("repo", repoURL), writeOutput(output, repoURL), repoURL, repoAuth, pattern); err != nil {
				log.Warnf("Error in %q: %s", repoURL, err)
			}
		}
		c++
	}

	if t.Err() != nil {
		// Loop through the errors to see the details
		for i, err := range t.Errs() {
			fmt.Printf("error #%d: %s", i, err)
		}
		log.Fatal(t.Err())
	}

	log.Println("count: ", c)

}

func matchPatterns(list []string, patterns ...string) []string {
	var matches []string
	for _, entry := range list {
		for _, pattern := range patterns {
			if strings.HasSuffix(entry, pattern) {
				matches = append(matches, pattern)
			}
		}
	}
	return matches
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

func waitForRemainingLimit(cl *github.Client, isCore bool, minLimit int) {
	for {
		rateLimits, _, err := cl.RateLimits(context.Background())
		if err != nil {
			log.Printf("could not access rate limit information: %s\n", err)
			<-time.After(time.Second * 1)
			continue
		}

		var rate int
		var limit int
		if isCore {
			rate = rateLimits.GetCore().Remaining
			limit = rateLimits.GetCore().Limit
		} else {
			rate = rateLimits.GetSearch().Remaining
			limit = rateLimits.GetSearch().Limit
		}

		if rate < minLimit {
			log.Printf("Not enough rate limit: %d/%d/%d\n", rate, minLimit, limit)
			<-time.After(time.Second * 60)
			continue
		}
		log.Printf("Rate limit: %d/%d\n", rate, limit)
		break
	}
}

func generateDateRange(query string, startYear, endYear int) (queries []string) {
	for curYear := endYear; curYear >= startYear; curYear-- {
		dateRange := fmt.Sprintf("created:%d-01-01..%d-12-31", curYear, curYear)
		queries = append(queries, fmt.Sprintf("%s %s", query, dateRange))
	}
	return
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
