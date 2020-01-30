package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v29/github"
	"github.com/k0kubun/pp"
	ghclient "github.com/x0rzkov/arXiv2git/golang/pkg/client"
	"go.uber.org/zap"
	// "gopkg.in/fatih/set.v0"
)

// Fetch all the public organizations' membership of a user.
//
func fetchGlobalTopics(topic string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var (
		// client *ghclient.GHClient
		ok      bool
		wg      sync.WaitGroup
		results []string
		e       *github.AbuseRateLimitError
	)
	log.Println("fetchGlobalTopics, topic=", topic)
	// topics, _, err := clientGH.Client.Search.Topics(context.Background(), topic, nil)

getGlobalTopics:
	checkForRemainingLimit(false, 10)
	topics, resp, err := clientGH.Client.Search.Topics(ctx, topic, nil /*&github.SearchOptions{}*/)
	if err != nil {
		if _, ok = err.(*github.RateLimitError); ok {
			log.Error("fetchGlobalTopics hit limit error, it's time to change client.", zap.Error(err))

			goto changeClient
		} else if e, ok = err.(*github.AbuseRateLimitError); ok {
			log.Error("fetchGlobalTopics have triggered an abuse detection mechanism.", zap.Error(err))
			time.Sleep(*e.RetryAfter)
			goto getGlobalTopics

		} else if strings.Contains(err.Error(), "timeout") {
			log.Info("fetchGlobalTopics has encountered a timeout error. Sleep for five minutes.")
			time.Sleep(5 * time.Minute)
			goto getGlobalTopics

		} else {
			log.Error("fetchGlobalTopics terminated because of this error.", zap.Error(err))
			return nil, err
		}
	}
	for _, result := range topics.Topics {
		results = append(results, *result.Name)
	}

	return results, nil

changeClient:
	{
		log.Warnln("getTopics2.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			ghclient.Reclaim(clientGH, resp)
		}()
		clientGH = clientManager.Fetch()
		goto getGlobalTopics
	}
	pp.Println("results: ", results)
	return results, nil
}

func getTopics(client *github.Client, owner, name string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	waitForRemainingLimit(client, true, 10)
	topics, _, err := client.Repositories.ListAllTopics(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return topics, nil
}

func getTopics2(owner, name string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var (
		// client *ghclient.GHClient
		ok     bool
		wg     sync.WaitGroup
		topics []string
		e      *github.AbuseRateLimitError
	)
	// defer clientManager.Shutdown()
	// client = clientManager.Fetch()
	log.Println("getTopics2, owner=", owner, "name=", name)

getTopics:
	checkForRemainingLimit(true, 10)
	topics, resp, err := clientGH.Client.Repositories.ListAllTopics(ctx, owner, name)
	if err != nil {
		if _, ok = err.(*github.RateLimitError); ok {
			log.Error("getTopics hit limit error, it's time to change client.", zap.Error(err))

			goto changeClient
		} else if e, ok = err.(*github.AbuseRateLimitError); ok {
			log.Error("getTopics have triggered an abuse detection mechanism.", zap.Error(err))
			time.Sleep(*e.RetryAfter)
			goto getTopics

		} else if strings.Contains(err.Error(), "timeout") {
			log.Info("getTopics has encountered a timeout error. Sleep for five minutes.")
			time.Sleep(5 * time.Minute)
			goto getTopics

		} else {
			log.Error("getTopics terminated because of this error.", zap.Error(err))
			return nil, err
		}
	}
	return topics, nil

changeClient:
	{
		log.Warnln("getTopics2.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			ghclient.Reclaim(clientGH, resp)
		}()
		clientGH = clientManager.Fetch()
		goto getTopics
	}
	pp.Println("topics: ", topics)
	return topics, nil
}

func getLanguages2(owner, name string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var (
		ok        bool
		wg        sync.WaitGroup
		langs     map[string]int
		languages []string
		e         *github.AbuseRateLimitError
	)
	// client = clientManager.Fetch()
	log.Println("getTopics2, owner=", owner, "name=", name)

getLanguages:
	checkForRemainingLimit(true, 10)
	langs, resp, err := clientGH.Client.Repositories.ListLanguages(ctx, owner, name)
	if err != nil {
		if _, ok = err.(*github.RateLimitError); ok {
			log.Error("getTopics hit limit error, it's time to change client.", zap.Error(err))

			goto changeClient
		} else if e, ok = err.(*github.AbuseRateLimitError); ok {
			log.Error("getTopics have triggered an abuse detection mechanism.", zap.Error(err))
			time.Sleep(*e.RetryAfter)
			goto getLanguages

		} else if strings.Contains(err.Error(), "timeout") {
			log.Info("getTopics has encountered a timeout error. Sleep for five minutes.")
			time.Sleep(5 * time.Minute)
			goto getLanguages

		} else {
			log.Error("getTopics terminated because of this error.", zap.Error(err))
			return nil, err
		}
	}
	for lang, _ := range langs {
		languages = append(languages, lang)
	}
	return languages, nil

changeClient:
	{
		log.Warnln("getLanguages2.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			ghclient.Reclaim(clientGH, resp)
		}()
		clientGH = clientManager.Fetch()
		goto getLanguages
	}

	return languages, nil
}

func getEntries(client *github.Client, owner, name, branch string, recursive bool) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	waitForRemainingLimit(client, true, 10)
	tree, _, err := client.Git.GetTree(ctx, owner, name, branch, recursive)
	if err != nil {
		return nil, err
	}
	var entries []string
	for _, entry := range tree.Entries {
		entries = append(entries, *entry.Path)
	}
	return entries, nil
}

func getEntries2(owner, name, branch string, recursive bool) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	var (
		// client  *ghclient.GHClient
		ok      bool
		wg      sync.WaitGroup
		entries []string
		e       *github.AbuseRateLimitError
	)
	log.Println("getEntries2x, owner=", owner, "name=", name, "branch=", branch)
	// defer clientManager.Shutdown()
	// client = clientManager.Fetch()
	// pp.Println("client: ", client)
	// log.Println("getEntries2y, owner=", owner, "name=", name, "branch=", branch)
	// goto getFiles

getFiles:
	checkForRemainingLimit(true, 10)
	tree, resp, err := clientGH.Client.Git.GetTree(ctx, owner, name, branch, recursive)
	if err != nil {
		if _, ok = err.(*github.RateLimitError); ok {
			log.Error("getEntries hit limit error, it's time to change client.", zap.Error(err))

			goto changeClient
		} else if e, ok = err.(*github.AbuseRateLimitError); ok {
			log.Error("getEntries have triggered an abuse detection mechanism.", zap.Error(err))
			time.Sleep(*e.RetryAfter)
			goto getFiles

		} else if strings.Contains(err.Error(), "timeout") {
			log.Info("getEntries has encountered a timeout error. Sleep for five minutes.")
			time.Sleep(5 * time.Minute)
			goto getFiles

		} else {
			log.Error("getEntries terminated because of this error.", zap.Error(err))
			return nil, err
		}
	}
	for _, entry := range tree.Entries {
		entries = append(entries, *entry.Path)
	}
	return entries, nil

changeClient:
	{
		log.Warnln("getEntries2.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			ghclient.Reclaim(clientGH, resp)
		}()
		clientGH = clientManager.Fetch()
		goto getFiles
	}

	return entries, nil
}

func getFileContent(client *github.Client, owner, repoName, branch, path string) (string, error) {
	waitForRemainingLimit(client, true, 10)
	content, _, resp, err := client.Repositories.GetContents(context.Background(), owner, repoName, path, &github.RepositoryContentGetOptions{Ref: branch})
	if resp.StatusCode != 200 {
		return "", errors.New("Bad response from Github: " + resp.Status)
	}
	if content == nil {
		return "", err
	}
	decoded, err := content.GetContent()
	if err != nil {
		return "", err
	}
	return decoded, nil
}

func getFileContent2(owner, repoName, branch, path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	var (
		// client *ghclient.GHClient
		ok bool
		wg sync.WaitGroup
		e  *github.AbuseRateLimitError
		// content string
		decoded string
	)

	// defer clientManager.Shutdown()
	// client = clientManager.Fetch()
	log.Println("getFileContent2, owner=", owner, "repoName=", repoName, "branch=", branch)

getFile:
	checkForRemainingLimit(true, 10)
	content, _, resp, err := clientGH.Client.Repositories.GetContents(ctx, owner, repoName, path, &github.RepositoryContentGetOptions{Ref: branch})
	if err != nil {
		if _, ok = err.(*github.RateLimitError); ok {
			log.Error("getEntries hit limit error, it's time to change client.", zap.Error(err))
			goto changeClient

		} else if e, ok = err.(*github.AbuseRateLimitError); ok {
			log.Error("getEntries have triggered an abuse detection mechanism.", zap.Error(err))
			time.Sleep(*e.RetryAfter)
			goto getFile

		} else if strings.Contains(err.Error(), "timeout") {
			log.Info("getEntries has encountered a timeout error. Sleep for five minutes.")
			time.Sleep(5 * time.Minute)
			goto getFile

		} else {
			log.Error("getEntries terminated because of this error.", zap.Error(err))
			return "", err
		}
	}
	if resp.StatusCode != 200 {
		return "", errors.New("Bad response from Github: " + resp.Status)
	}
	if content == nil {
		return "", err
	}
	decoded, err = content.GetContent()
	if err != nil {
		return "", err
	}
	return decoded, nil

changeClient:
	{
		log.Warnln("getFileContent2.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			ghclient.Reclaim(clientGH, resp)
		}()
		clientGH = clientManager.Fetch()
		goto getFile
	}

	return decoded, nil
}

func getReadme(client *github.Client, owner, repoName string) (string, error) {
	waitForRemainingLimit(client, true, 10)
	readme, _, err := client.Repositories.GetReadme(context.Background(), owner, repoName, nil)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	content, err := readme.GetContent()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return content, nil
}

func getReadme2(owner, repoName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var (
		// client  *ghclient.GHClient
		ok      bool
		wg      sync.WaitGroup
		content string
		e       *github.AbuseRateLimitError
	)
	// defer clientManager.Shutdown()
	// client = clientManager.Fetch()

	log.Println("getReadme2, owner=", owner, "repoName=", repoName)

getReadme:
	checkForRemainingLimit(true, 10)
	readme, resp, err := clientGH.Client.Repositories.GetReadme(ctx, owner, repoName, nil)
	if err != nil {
		if _, ok = err.(*github.RateLimitError); ok {
			log.Error("getEntries hit limit error, it's time to change client.", zap.Error(err))

			goto changeClient
		} else if e, ok = err.(*github.AbuseRateLimitError); ok {
			log.Error("getEntries have triggered an abuse detection mechanism.", zap.Error(err))
			time.Sleep(*e.RetryAfter)
			goto getReadme

		} else if strings.Contains(err.Error(), "timeout") {
			log.Info("getEntries has encountered a timeout error. Sleep for five minutes.")
			time.Sleep(5 * time.Minute)
			goto getReadme

		} else {
			log.Error("getEntries terminated because of this error.", zap.Error(err))
			return "", err
		}
	}
	content, err = readme.GetContent()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return content, nil

changeClient:
	{
		log.Warnln("getReadme2.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			ghclient.Reclaim(clientGH, resp)
		}()
		clientGH = clientManager.Fetch()
		goto getReadme
	}
	return content, nil
}

func listBranches(client *github.Client, owner, repoName string) ([]string, error) {
	var (
		branchNames []string
		resp        = new(github.Response)
		listOpts    = &github.BranchListOptions{
			Protected: nil,
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}
	)

	waitForRemainingLimit(client, true, 10)
	// Get all pages
	resp.NextPage = 1
	for resp.NextPage != 0 {
		listOpts.Page = resp.NextPage
		fetched, newResp, err := client.Repositories.ListBranches(context.Background(), owner, repoName, listOpts)
		resp = newResp
		if err != nil {
			return nil, err
		}
		for _, branch := range fetched {
			branchNames = append(branchNames, *branch.Name)
		}
	}
	return branchNames, nil
}

func listBranches2(owner, repoName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var (
		// client      *ghclient.GHClient
		ok          bool
		wg          sync.WaitGroup
		e           *github.AbuseRateLimitError
		branchNames []string
		resp        = new(github.Response)
		listOpts    = &github.BranchListOptions{
			Protected: nil,
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}
	)

	// defer clientManager.Shutdown()
	// client = clientManager.Fetch()
	// pp.Println("client: ", client)

	log.Println("listBranches2, owner=", owner, "repoName=", repoName)

getBranch:
	// Get all pages
	resp.NextPage = 1
	for resp.NextPage != 0 {
		listOpts.Page = resp.NextPage
		checkForRemainingLimit(true, 10)
		fetched, newResp, err := clientGH.Client.Repositories.ListBranches(ctx, owner, repoName, listOpts)
		resp = newResp
		if err != nil {
			if _, ok = err.(*github.RateLimitError); ok {
				log.Error("getEntries hit limit error, it's time to change client.", zap.Error(err))

				goto changeClient
			} else if e, ok = err.(*github.AbuseRateLimitError); ok {
				log.Error("getEntries have triggered an abuse detection mechanism.", zap.Error(err))
				time.Sleep(*e.RetryAfter)
				goto getBranch

			} else if strings.Contains(err.Error(), "timeout") {
				log.Info("getEntries has encountered a timeout error. Sleep for five minutes.")
				time.Sleep(5 * time.Minute)
				goto getBranch

			} else {
				log.Error("getEntries terminated because of this error.", zap.Error(err))
				return nil, err
			}
		}
		for _, branch := range fetched {
			branchNames = append(branchNames, *branch.Name)
		}
		pp.Println("branchNames:", branchNames)
	}
	return branchNames, nil

changeClient:
	{
		log.Warnln("listBranches2.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			ghclient.Reclaim(clientGH, resp)
		}()
		clientGH = clientManager.Fetch()
		goto getBranch
	}

	return branchNames, nil
}

func getAllRepositories(client *github.Client, organization string) ([]*github.Repository, error) {
	var (
		repositories []*github.Repository
		resp         = new(github.Response)
		listOpts     = &github.RepositoryListByOrgOptions{
			Type: "sources",
			Sort: "created",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}
	)
	waitForRemainingLimit(client, true, 10)
	// Get all pages
	resp.NextPage = 1
	for resp.NextPage != 0 {
		listOpts.Page = resp.NextPage
		fetched, newResp, err := client.Repositories.ListByOrg(context.Background(), organization, listOpts)
		resp = newResp
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, fetched...)

	}
	return repositories, nil
}

/*
func processRepository(client *github.Client, repository *github.Repository, excludedBranches []string, dryRun bool) error {
	var (
		owner    = *repository.Owner.Login
		repoName = *repository.Name
	)
	// Collect branches than are currently in use as target or source branch in open PRs, to avoid deleting them
	openPRs, err := pullRequestsByState(client, owner, repoName, "open")
	if err != nil {
		return err
	}
	excluded := buildExclusionList(excludedBranches, openPRs)

	// Collect all closed PRs to scan
	closedPRs, err := pullRequestsByState(client, owner, repoName, "closed")
	if err != nil {
		return err
	}

	// Collect all existing branches, try not to delete already deleted branches
	existingBranches, err := listBranches(client, owner, repoName)
	if err != nil {
		return err
	}

	for _, closedPR := range closedPRs {
		var (
			sourceBranch = *closedPR.Head.Ref
			sourceRepo   = *closedPR.Head.User.Login
		)
		for _, branch := range existingBranches {
			// Delete if:
			// -> the old source branch matches an existing source branch
			// -> the source branch was on the same repository (don't touch forks, leave it to jessfraz/ghb0t)
			// -> the branch is not in the exclusion list
			if branch == sourceBranch && sourceRepo == owner && !excluded.Has(sourceBranch) {
				if !dryRun {
					if _, err := client.Git.DeleteRef(context.Background(), owner, repoName, fmt.Sprintf("refs/%s", sourceBranch)); err != nil {
						return err
					}
				}
				fmt.Printf("%s/%s#%d => unused branch %s deleted.\n", owner, repoName, *closedPR.Number, sourceBranch)

			}
		}
	}
	return nil
}
*/

func pullRequestsByState(client *github.Client, owner string, repoName string, state string) ([]*github.PullRequest, error) {
	var (
		pullRequests []*github.PullRequest
		resp         = new(github.Response)
		listOpts     = &github.PullRequestListOptions{state, "", "", "", "", github.ListOptions{PerPage: 100}}
	)
	waitForRemainingLimit(client, true, 10)
	// Get all pages
	resp.NextPage = 1
	for resp.NextPage != 0 {
		listOpts.Page = resp.NextPage
		fetched, newResp, err := client.PullRequests.List(context.Background(), owner, repoName, listOpts)
		resp = newResp
		if err != nil {
			return nil, err
		}
		pullRequests = append(pullRequests, fetched...)

	}
	return pullRequests, nil
}

/*
func buildExclusionList(excludedBranches []string, openPRs []*github.PullRequest) *set.SetNonTS {
	excluded := set.New()
	for _, branch := range excludedBranches {
		excluded.Add(branch)
	}
	for _, openPR := range openPRs {
		excluded.Add(*openPR.Base.Ref)
		excluded.Add(*openPR.Head.Ref)
	}
	return excluded
}
*/
