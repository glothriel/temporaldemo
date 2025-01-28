package github

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// This is very simplified approach, only used to demonstrate the idea
type Branch string

type Repo interface {
	Head() Branch

	Create(Branch) error
	Checkout(Branch) error
	Delete(Branch) error
	Merge(Branch) error
	Push(context.Context, Branch) error
}

type PullRequestID string

type Client interface {
	CreatePullRequest(ctx context.Context, repo Repo, base, head Branch) (PullRequestID, error)
	MergePullRequest(ctx context.Context, repo Repo, prID PullRequestID) error
}

type MockRepo struct {
}

func (r *MockRepo) Head() Branch {
	return "main"
}

func (r *MockRepo) Create(b Branch) error {
	logrus.Warnf("Created %s", b)
	return nil
}

func (r *MockRepo) Checkout(b Branch) error {
	logrus.Warnf("Checked out %s", b)
	return nil
}

func (r *MockRepo) Delete(b Branch) error {
	logrus.Warnf("Deleted out %s", b)
	return nil
}

func (r *MockRepo) Merge(b Branch) error {

	logrus.Warnf("Checked out %s to %s", b, r.Head())
	return nil
}

func (r *MockRepo) Push(_ context.Context, b Branch) error {
	logrus.Warnf("Pushed %s", b)
	return nil
}

type MockClient struct {
}

func (c *MockClient) CreatePullRequest(_ context.Context, r Repo, base Branch, target Branch) (PullRequestID, error) {
	logrus.Warnf("Created PR from %s to %s", base, target)
	return "123", nil
}

func (c *MockClient) MergePullRequest(_ context.Context, r Repo, prID PullRequestID) error {
	logrus.Warnf("Merged PR %s", prID)
	return nil
}

type ReleaseProcess struct {
	Repo       Repo
	Client     Client
	BaseBranch Branch
}

func (r *ReleaseProcess) PrepareAndPushReleaseBranch(ctx context.Context, release string) (Branch, error) {
	if checkoutErr := r.Repo.Checkout(r.BaseBranch); checkoutErr != nil {
		return "", fmt.Errorf("failed to checkout base branch: %w", checkoutErr)
	}
	releaseBranch := Branch(fmt.Sprintf("release/%s", release))
	if createErr := r.Repo.Create(releaseBranch); createErr != nil {
		return "", fmt.Errorf("failed to create release branch: %w", createErr)
	}
	if mergeErr := r.Repo.Merge(r.BaseBranch); mergeErr != nil {
		return "", fmt.Errorf("failed to merge base branch: %w", mergeErr)
	}
	if pushErr := r.Repo.Push(ctx, releaseBranch); pushErr != nil {
		return "", fmt.Errorf("failed to push release branch: %w", pushErr)
	}
	return releaseBranch, nil
}

func (r *ReleaseProcess) CreatePR(ctx context.Context, releaseBranch Branch) (PullRequestID, error) {
	return r.Client.CreatePullRequest(ctx, r.Repo, r.BaseBranch, releaseBranch)
}

func (r *ReleaseProcess) MergePR(ctx context.Context, prID PullRequestID) error {
	return r.Client.MergePullRequest(ctx, r.Repo, prID)
}

func (r *ReleaseProcess) DeleteReleaseBranch(ctx context.Context, releaseBranch Branch) error {
	return r.Repo.Delete(releaseBranch)
}

func process(r Repo, c Client, release string) {
	r.Checkout("master")
	releaseBranch := Branch(fmt.Sprintf("release/%s", release))
	r.Create(releaseBranch)
	r.Merge("develop")
	r.Push(context.Background(), releaseBranch)
	prId, _ := c.CreatePullRequest(context.Background(), r, r.Head(), "master")

	// Wait here for the PR to be approved

	c.MergePullRequest(context.Background(), r, prId)
	r.Delete(releaseBranch)
	r.Push(context.Background(), releaseBranch)
}
