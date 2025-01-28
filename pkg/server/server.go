package server

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glothriel/tempogo/pkg/github"
	"github.com/sirupsen/logrus"
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

	logrus.Info("Starting Temporal client")
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	logrus.Info("Temporal client started")
	defer c.Close()

	r.POST("/", func(ctx *gin.Context) {
		var createInput createRequest
		if err := ctx.BindJSON(&createInput); err != nil || createInput.ReleaseName == "" {
			ctx.JSON(400, gin.H{
				"error": "Invalid request",
			})
			return
		}

		workflowID := "release/" + createInput.ReleaseName + "-" + time.Now().Format("2006-01-02T15:04:05")

		options := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: github.QueueName,
		}

		wf, err := c.ExecuteWorkflow(context.Background(), options, github.OrchestrateReleaseProcess, createInput.ReleaseName)
		if err != nil {
			log.Fatalln("Unable to execute workflow", err)
		}

		ctx.JSON(200, gin.H{
			"result": wf.GetID(),
		})
	})

	r.POST("/approve", func(ctx *gin.Context) {
		// Parse the request
		var approveInput workloadIDRequest
		if err := ctx.BindJSON(&approveInput); err != nil {
			ctx.JSON(400, gin.H{
				"error": "Invalid request",
			})
			return
		}

		// Signal the workflow
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
		// Parse the request
		var stopInput workloadIDRequest
		if err := ctx.BindJSON(&stopInput); err != nil || stopInput.WorkflowID == "" {
			ctx.JSON(400, gin.H{
				"error": "Invalid request",
			})
			return
		}

		// Cancel the workflow
		logrus.Infof("Cancelling workflow %s", stopInput.WorkflowID)
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
