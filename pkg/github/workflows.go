package github

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// OrchestrateReleaseProcess is the Temporal Workflow that retrieves the IP address and location info.
func OrchestrateReleaseProcess(ctx workflow.Context, releaseName string) (err error) {
	// Define the activity options, including the retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,      //amount of time that must elapse before the first retry occurs
			MaximumInterval:    time.Second * 10, //maximum interval between retries
			BackoffCoefficient: 1.1,              //how much the retry interval increases
		},
	}
	saga := NewSaga()

	defer func() {
		if err != nil {
			disconnectedCtx, _ := workflow.NewDisconnectedContext(ctx)
			saga.Unwind(disconnectedCtx)
		}
	}()
	ctx = workflow.WithActivityOptions(ctx, ao)

	var rp *ReleaseProcess

	err = workflow.ExecuteActivity(ctx, rp.PrepareAndPushReleaseBranch, releaseName).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("Failed to prepare and push release branch: %s", err)
	}
	saga.Add(rp.DeleteReleaseBranch, releaseName)

	var prID PullRequestID
	err = workflow.ExecuteActivity(ctx, rp.CreatePR, releaseName).Get(ctx, &prID)
	if err != nil {
		return fmt.Errorf("Failed to create pull request: %s", err)
	}
	saga.Add(rp.DeletePR, prID)

	sc := workflow.GetSignalChannel(ctx, ApproveSignal)

	err = workflow.Await(ctx, func() bool {
		var approveInput any
		return sc.ReceiveAsync(&approveInput)
	})
	if err != nil {
		return fmt.Errorf("Failed to receive approval: %s", err)
	}
	logrus.Warn("Received approval")

	err = workflow.ExecuteActivity(ctx, rp.MergePR, prID).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("Failed to merge pull request: %s", err)
	}

	err = workflow.ExecuteActivity(ctx, rp.TagRelease, releaseName).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("Failed to delete release branch: %s", err)
	}

	err = workflow.ExecuteActivity(ctx, rp.DeleteReleaseBranch, releaseName).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("Failed to delete release branch: %s", err)
	}

	return nil
}

func WaitUntilPRIsAccepted(ctx workflow.Context, pullRequestID string) (err error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,      //amount of time that must elapse before the first retry occurs
			MaximumInterval:    time.Second * 10, //maximum interval between retries
			BackoffCoefficient: 1.1,              //how much the retry interval increases
		},
	}
	var rp *ReleaseProcess
	ctx = workflow.WithActivityOptions(ctx, ao)
	isAccepted := false

	for !isAccepted {
		err = workflow.ExecuteActivity(ctx, rp.IsPullRequestAccepted, pullRequestID).Get(ctx, &isAccepted)
		if err != nil {
			return fmt.Errorf("Failed to check if PR is accepted: %s", err)
		}
		logrus.Infof("Checking if PR is accepted (%v)", isAccepted)
		if !isAccepted {
			sleepErr := workflow.Sleep(ctx, time.Second)
			if sleepErr != nil {
				return fmt.Errorf("Failed to sleep: %s", sleepErr)
			}
		}
	}
	return nil
}
