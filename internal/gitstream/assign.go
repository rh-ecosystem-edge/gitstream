package gitstream

import (
	"bytes"
	"context"
	"fmt"
	"path"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v47/github"
	"github.com/hashicorp/go-multierror"
	"github.com/rh-ecosystem-edge/gitstream/internal"
	"github.com/rh-ecosystem-edge/gitstream/internal/config"
	gh "github.com/rh-ecosystem-edge/gitstream/internal/github"
	"github.com/rh-ecosystem-edge/gitstream/internal/gitutils"
	"github.com/rh-ecosystem-edge/gitstream/internal/markup"
	"github.com/rh-ecosystem-edge/gitstream/internal/owners"
)

type Assign struct {
	GC               *github.Client
	DryRun           bool
	Finder           markup.Finder
	GitHelper        gitutils.Helper
	Logger           logr.Logger
	IssueHelper      gh.IssueHelper
	UserHelper       gh.UserHelper
	Repo             *git.Repository
	RepoName         *gh.RepoName
	UpstreamConfig   config.Upstream
	DownstreamConfig config.Downstream
	OwnersHelper     owners.OwnersHelper
}

func (a *Assign) Run(ctx context.Context) error {
	const remoteName = internal.UpstreamRemoteName

	if _, err := a.GitHelper.RecreateRemote(ctx, remoteName, a.UpstreamConfig.URL); err != nil {
		return fmt.Errorf("could not recreate remote: %v", err)
	}

	if err := a.GitHelper.FetchRemoteContext(ctx, remoteName, a.UpstreamConfig.Ref); err != nil {
		return fmt.Errorf("could not fetch remote %s: %v", remoteName, err)
	}

	if err := a.assignIssues(ctx); err != nil {
		return fmt.Errorf("could not add assignees to issues: %v", err)
	}

	return nil
}

func (a *Assign) filterApproversFromCommitAuthors(commitAuthors []string, owners *owners.Owners) []string {

	filteredCommitAuthors := make([]string, 0, len(commitAuthors))
	for _, ca := range commitAuthors {
		if a.OwnersHelper.IsApprover(owners, ca) {
			filteredCommitAuthors = append(filteredCommitAuthors, ca)
		}
	}

	return filteredCommitAuthors
}

func (a *Assign) handleIssue(ctx context.Context, issue *github.Issue, owners *owners.Owners) error {

	logger := a.Logger.WithValues("url", *issue.HTMLURL, "issue", *issue.Number)

	if len(issue.Assignees) > 0 {
		return nil
	}

	logger.Info("Processing issue")

	shas, err := a.Finder.FindSHAs(*issue.Body)
	if err != nil {
		return fmt.Errorf("error while looking for SHAs in %q: %v", *issue.Body, err)
	}

	// Convert SHAs to strings for the comment
	shaStrings := make([]string, len(shas))
	for i, s := range shas {
		shaStrings[i] = s.String()
	}

	commitAuthors := make([]string, 0, len(shas))
	for _, s := range shas {
		user, err := a.UserHelper.GetCommitAuthor(ctx, s.String())
		if err != nil {
			return fmt.Errorf("failed to get commit author from GitHub for issue %d in commit %s: %v",
				*issue.Number, s.String(), err)
		}
		commitAuthors = append(commitAuthors, *user.Login)
	}

	approverCommitAuthors := a.filterApproversFromCommitAuthors(commitAuthors, owners)
	assignees := approverCommitAuthors

	var isRandomAssignment bool
	var assignmentReason string

	if len(assignees) == 0 {
		logger.Info("None of the commit authors are approvers, picking a random approver")
		randAssignee, err := a.OwnersHelper.GetRandomApprover(owners)
		if err != nil {
			return fmt.Errorf("could not get a random approver: %v", err)
		}
		assignees = append(assignees, randAssignee)
		isRandomAssignment = true
		assignmentReason = "none of the commit authors are approvers in the OWNERS file."
	} else {
		isRandomAssignment = false
		if len(approverCommitAuthors) == 1 {
			assignmentReason = "they are the author of a referenced commit and an approver."
		} else {
			assignmentReason = "they are authors of referenced commits and approvers."
		}
	}

	if err := a.IssueHelper.Assign(ctx, issue, assignees...); err != nil {
		return fmt.Errorf("could not assign issue %d to %s: %v", *issue.Number, assignees, err)
	}

	// Create assignment comment if not in dry run mode
	if !a.DryRun {
		commentData := gh.AssignmentCommentData{
			AppName:               internal.AppName,
			CommitSHAs:            shaStrings,
			CommitAuthors:         commitAuthors,
			ApproverCommitAuthors: approverCommitAuthors,
			AssignedUsers:         assignees,
			AssignmentReason:      assignmentReason,
			IsRandomAssignment:    isRandomAssignment,
		}

		if err := a.createAssignmentComment(ctx, issue, &commentData); err != nil {
			// Log the error but don't fail the assignment
			logger.Error(err, "Failed to create assignment comment")
		}
	}

	return nil
}

func (a *Assign) assignIssues(ctx context.Context) error {

	ownersFile := path.Join(a.DownstreamConfig.LocalRepoPath, a.DownstreamConfig.OwnersFile)
	owners, err := a.OwnersHelper.FromFile(ownersFile)
	if err != nil {
		return fmt.Errorf("could not get owners from file %s: %v", ownersFile, err)
	}

	issues, err := a.IssueHelper.ListAllOpen(ctx, true)
	if err != nil {
		return fmt.Errorf("could not list open issues: %v", err)
	}

	var multiErr error
	for _, issue := range issues {
		if err := a.handleIssue(ctx, issue, owners); err != nil {
			multiErr = multierror.Append(multiErr, err)
		}
	}

	return multiErr
}

func (a *Assign) createAssignmentComment(ctx context.Context, issue *github.Issue, data *gh.AssignmentCommentData) error {
	var buf bytes.Buffer

	if err := gh.ExecuteAssignmentCommentTemplate(&buf, data); err != nil {
		return fmt.Errorf("could not execute assignment comment template: %v", err)
	}

	if err := a.IssueHelper.Comment(ctx, issue, buf.String()); err != nil {
		return fmt.Errorf("could not create assignment comment: %v", err)
	}

	return nil
}
