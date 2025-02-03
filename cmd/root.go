package cmd

import (
	"context"

	"github.com/glothriel/tempogo/pkg/github"
	"github.com/glothriel/tempogo/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func Start(args []string) error {
	return (&cli.Command{
		Commands: []*cli.Command{
			{
				Name:  "server",
				Usage: "Start HTTP server",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return server.Start()
				},
			},
			{
				Name:  "worker",
				Usage: "Start a Temporal worker",
				Action: func(ctx context.Context, _ *cli.Command) error {
					c, dialErr := client.Dial(client.Options{})
					if dialErr != nil {
						logrus.Panicf("Unable to connect to Temporal server: %v", dialErr)
					}
					defer c.Close()

					w := worker.New(c, github.QueueName, worker.Options{})
					w.RegisterWorkflow(github.OrchestrateReleaseProcess)
					w.RegisterActivity(&github.ReleaseProcess{
						Client:     &github.MockClient{},
						Repo:       &github.MockRepo{},
						BaseBranch: "master",
					})

					runErr := w.Run(worker.InterruptCh())
					if runErr != nil {
						logrus.Panicf("Unable to start worker: %v", runErr)
					}
					return nil
				},
			},
		}}).Run(context.Background(), args)
}
