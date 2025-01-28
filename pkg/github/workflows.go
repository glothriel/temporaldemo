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
			InitialInterval:    time.Second, //amount of time that must elapse before the first retry occurs
			MaximumInterval:    time.Minute, //maximum interval between retries
			BackoffCoefficient: 2,           //how much the retry interval increases
			// MaximumAttempts: 5, // Uncomment this if you want to limit attempts
		},
	}
	saga := NewSaga()

	defer func() {
		if err != nil {
			disconnectedCtx, _ := workflow.NewDisconnectedContext(ctx)
			saga.Add(disconnectedCtx, true)
		}
	}()
	ctx = workflow.WithActivityOptions(ctx, ao)

	var rp *ReleaseProcess

	var releaseBranch Branch
	err = workflow.ExecuteActivity(ctx, rp.PrepareAndPushReleaseBranch, releaseName).Get(ctx, &releaseBranch)
	if err != nil {
		return fmt.Errorf("Failed to prepare and push release branch: %s", err)
	}
	saga.Unwind(rp.DeleteReleaseBranch, releaseBranch)

	err = workflow.ExecuteActivity(ctx, rp.CreatePR, releaseBranch).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("Failed to create pull request: %s", err)
	}

	sc := workflow.GetSignalChannel(ctx, ApproveSignal)
	err = workflow.Await(ctx, func() bool {
		var approveInput any
		return sc.ReceiveAsync(&approveInput)
	})
	if err != nil {
		return fmt.Errorf("Failed to receive approval: %s", err)
	}
	logrus.Warn("Received approval")

	err = workflow.ExecuteActivity(ctx, rp.MergePR, releaseBranch).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("Failed to merge pull request: %s", err)
	}

	err = workflow.ExecuteActivity(ctx, rp.DeleteReleaseBranch, releaseBranch).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("Failed to delete release branch: %s", err)
	}

	return nil
}
