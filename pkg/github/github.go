package github

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const QueueName = "github"
const ApproveSignal = "approve"

// This is very simplified approach, only used to demonstrate the idea
type RefName string
type Ref string

type Repo interface {
	Head() RefName

	Create(RefName) error
	Checkout(RefName) error
	Delete(RefName) error
	Merge(RefName) error
	Push(context.Context, Ref) error
}

type PullRequestID string

type Client interface {
	CreatePullRequest(ctx context.Context, repo Repo, base, from RefName) (PullRequestID, error)
	MergePullRequest(ctx context.Context, repo Repo, prID PullRequestID) error
	DeletePullRequest(ctx context.Context, repo Repo, prID PullRequestID) error
	IsPullRequestAccepted(ctx context.Context, repo Repo, prID PullRequestID) (bool, error)
}

type MockRepo struct {
}

func (r *MockRepo) Head() RefName {
	return "master"
}

func (r *MockRepo) Create(b RefName) error {
	logrus.Warnf("Created %s", b)
	return nil
}

func (r *MockRepo) Checkout(b RefName) error {
	logrus.Warnf("Checked out %s", b)
	return nil
}

func (r *MockRepo) Delete(b RefName) error {
	logrus.Warnf("Deleted %s", b)
	return nil
}

func (r *MockRepo) Merge(b RefName) error {

	logrus.Warnf("Merged %s to %s", b, r.Head())
	return nil
}

func (r *MockRepo) Push(_ context.Context, b Ref) error {
	logrus.Warnf("Pushed %s", b)
	return nil
}

type MockClient struct {
}

func (c *MockClient) CreatePullRequest(_ context.Context, r Repo, base RefName, from RefName) (PullRequestID, error) {
	prID := uuid.New().String()
	logrus.Warnf("Created PR ID %s from %s to %s", prID, from, base)
	return PullRequestID(prID), nil
}

func (c *MockClient) MergePullRequest(_ context.Context, r Repo, prID PullRequestID) error {
	logrus.Warnf("Merged PR %s", prID)
	return nil
}

func (c *MockClient) DeletePullRequest(_ context.Context, r Repo, prID PullRequestID) error {
	logrus.Warnf("Deleted PR %s", prID)
	return nil
}
func (c *MockClient) IsPullRequestAccepted(_ context.Context, r Repo, prID PullRequestID) (bool, error) {
	_, statErr := os.Stat(path.Join("/tmp", string(prID)))
	isAccepted := true
	if statErr != nil {
		isAccepted = false
	}
	logrus.Warnf("Checking if PR is accepted: %v (%v)", isAccepted, statErr)
	return isAccepted, nil
}

type ReleaseProcess struct {
	Repo       Repo
	Client     Client
	BaseBranch RefName
}

func (r *ReleaseProcess) PrepareAndPushReleaseBranch(ctx context.Context, release string) (RefName, error) {
	if checkoutErr := r.Repo.Checkout(r.BaseBranch); checkoutErr != nil {
		return "", fmt.Errorf("failed to checkout base branch: %w", checkoutErr)
	}
	releaseName := RefName(fmt.Sprintf("release/%s", release))
	if createErr := r.Repo.Create(releaseName); createErr != nil {
		return "", fmt.Errorf("failed to create release branch: %w", createErr)
	}
	if mergeErr := r.Repo.Merge("develop"); mergeErr != nil {
		return "", fmt.Errorf("failed to merge base branch: %w", mergeErr)
	}
	if pushErr := r.Repo.Push(ctx, Ref(fmt.Sprintf(
		"refs/heads/%s", releaseName,
	))); pushErr != nil {
		return "", fmt.Errorf("failed to push release branch: %w", pushErr)
	}
	return releaseName, nil
}

func ErrorOut(howProbable uint32) bool {
	return uuid.New().ID()%howProbable != 0
}

func (r *ReleaseProcess) CreatePR(ctx context.Context, release RefName) (PullRequestID, error) {
	return r.Client.CreatePullRequest(ctx, r.Repo, r.BaseBranch, RefName(fmt.Sprintf("release/%s", release)))
}

func (r *ReleaseProcess) MergePR(ctx context.Context, prID PullRequestID) error {
	return r.Client.MergePullRequest(ctx, r.Repo, prID)
}

func (r *ReleaseProcess) DeletePR(ctx context.Context, prID PullRequestID) error {
	return r.Client.DeletePullRequest(ctx, r.Repo, prID)
}

func (r *ReleaseProcess) TagRelease(ctx context.Context, releaseName RefName) error {
	return r.Repo.Push(ctx,
		Ref(fmt.Sprintf("refs/tags/%s", releaseName)),
	)
}

func (r *ReleaseProcess) IsPullRequestAccepted(ctx context.Context, prID PullRequestID) (bool, error) {
	return r.Client.IsPullRequestAccepted(ctx, r.Repo, prID)
}

func (r *ReleaseProcess) DeleteReleaseBranch(ctx context.Context, releaseName RefName) error {
	return r.Repo.Delete(
		RefName(fmt.Sprintf("release/%s", releaseName)),
	)
}
