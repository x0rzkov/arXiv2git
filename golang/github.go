package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-github/v28/github"
	// "gopkg.in/fatih/set.v0"
)

func getTopics(client *github.Client, owner, name string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	waitForRemainingLimit(client, true, 10)
	topics, _, err := client.Repositories.ListAllTopics(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return topics, nil
}

func getEntries(client *github.Client, owner, name, branch string, recursive bool) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

func getAllRepositories(client *github.Client, organization string) ([]*github.Repository, error) {
	var (
		repositories []*github.Repository
		resp         = new(github.Response)
		listOpts     = &github.RepositoryListByOrgOptions{"sources", github.ListOptions{PerPage: 100}}
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

func getFileContent(client *github.Client, owner, repoName, branch, path string) (string, error) {
	waitForRemainingLimit(client, true, 10)
	content, _, resp, err := client.Repositories.GetContents(context.Background(), owner, repoName, path, &github.RepositoryContentGetOptions{Ref: branch})
	if err != nil {
		return "", err
	}
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

func listBranches(client *github.Client, owner, repoName string) ([]string, error) {
	var (
		branchNames []string
		resp        = new(github.Response)
		listOpts    = &github.ListOptions{PerPage: 100}
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
