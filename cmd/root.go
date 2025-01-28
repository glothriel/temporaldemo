package cmd

import (
	"context"
	"log"

	"github.com/glothriel/tempogo/pkg/github"
	"github.com/glothriel/tempogo/pkg/server"
	"github.com/urfave/cli/v3"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func Start(args []string) error {
	return (&cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Fetch IP address and location information",
				Action: func(ctx context.Context, _ *cli.Command) error {
					server.Start()

					return nil
				},
			},
			{
				Name:  "worker",
				Usage: "Start a Temporal worker",
				Action: func(ctx context.Context, _ *cli.Command) error {
					// Create the Temporal client
					c, err := client.Dial(client.Options{})
					if err != nil {
						log.Fatalln("Unable to create Temporal client", err)
					}
					defer c.Close()

					// Create the Temporal worker
					w := worker.New(c, github.QueueName, worker.Options{})

					// inject HTTP client into the Activities Struct
					activities := &github.ReleaseProcess{
						Client: &github.MockClient{},
						Repo:   &github.MockRepo{},
					}

					// Register Workflow and Activities
					w.RegisterWorkflow(github.OrchestrateReleaseProcess)
					w.RegisterActivity(activities)

					// Start the Worker
					err = w.Run(worker.InterruptCh())
					if err != nil {
						log.Fatalln("Unable to start Temporal worker", err)
					}
					return nil
				},
			},
		}}).Run(context.Background(), args)
}
