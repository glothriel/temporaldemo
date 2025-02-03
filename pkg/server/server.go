package server

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/glothriel/temporaldemo/pkg/github"
	"github.com/sirupsen/logrus"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

type workloadIDRequest struct {
	WorkflowID string `json:"workflow_id"`
}

type createRequest struct {
	ReleaseName string `json:"release_name"`
}

func Start() error {
	r := gin.Default()

	c, dialErr := client.Dial(client.Options{})
	if dialErr != nil {
		logrus.Panicf("Unable to create client: %s", dialErr)
	}
	defer c.Close()

	r.GET("/", func(ctx *gin.Context) {
		list, listErr := c.ListWorkflow(
			context.Background(),
			&workflowservice.ListWorkflowExecutionsRequest{
				Namespace: "default",
				Query:     "WorkflowType='OrchestrateReleaseProcess'",
			},
		)
		if listErr != nil {
			logrus.Errorf("Unable to list workflows: %s", listErr)
			ctx.JSON(500, gin.H{
				"error": "Unable to list workflows",
			})
			return
		}
		var workflows []map[string]any
		for _, wf := range list.Executions {
			workflows = append(workflows, map[string]any{
				"workflow_id": wf.GetExecution().GetWorkflowId(),
				"status":      wf.GetStatus().String(),
			})
		}
		ctx.JSON(200, gin.H{
			"result": workflows,
		})
	})

	r.POST("/", func(ctx *gin.Context) {
		var createInput createRequest
		if err := ctx.BindJSON(&createInput); err != nil || createInput.ReleaseName == "" {
			ctx.JSON(400, gin.H{
				"error": "Invalid request",
			})
			return
		}

		wf, wfErr := c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
			ID:                    fmt.Sprintf("create-release-%s", createInput.ReleaseName),
			TaskQueue:             github.QueueName,
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		}, github.OrchestrateReleaseProcess, createInput.ReleaseName)
		if wfErr != nil {
			logrus.Errorf("Unable to create workflow: %s", wfErr)
			ctx.JSON(500, gin.H{
				"error": "Unable to create workflow",
			})
			return
		}

		ctx.JSON(200, gin.H{
			"result": wf.GetID(),
		})
	})

	r.POST("/approve", func(ctx *gin.Context) {
		var approveInput workloadIDRequest
		if err := ctx.BindJSON(&approveInput); err != nil {
			ctx.JSON(400, gin.H{
				"error": "Invalid request",
			})
			return
		}

		signalErr := c.SignalWorkflow(context.Background(), approveInput.WorkflowID, "", github.ApproveSignal, nil)
		if signalErr != nil {
			logrus.Errorf("Unable to signal workflow %s: %s", approveInput.WorkflowID, signalErr)
			ctx.JSON(500, gin.H{
				"error": "Unable to signal workflow",
			})
			return
		}
		ctx.Status(204)
	})

	r.POST("/cancel", func(ctx *gin.Context) {
		var stopInput workloadIDRequest
		if err := ctx.BindJSON(&stopInput); err != nil || stopInput.WorkflowID == "" {
			ctx.JSON(400, gin.H{
				"error": "Invalid request",
			})
			return
		}

		cancelErr := c.CancelWorkflow(context.Background(), stopInput.WorkflowID, "")
		if cancelErr != nil {
			logrus.Errorf("Unable to cancel workflow %s: %s", stopInput.WorkflowID, cancelErr)
			ctx.JSON(500, gin.H{
				"error": "Unable to cancel workflow",
			})
			return
		}
		ctx.Status(204)
	})

	return r.Run("127.0.0.1:9090")
}
