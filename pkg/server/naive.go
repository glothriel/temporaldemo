package server

import (
	"github.com/gin-gonic/gin"
	"github.com/glothriel/temporaldemo/pkg/github"
)

func RunSimpleServer() error {
	r := gin.Default()

	ghClient := &github.MockClient{}
	ghRepo := github.MockRepo{}
	p := github.ReleaseProcess{
		Client:     ghClient,
		Repo:       &ghRepo,
		BaseBranch: "master",
	}

	r.POST("/create/:release", func(ctx *gin.Context) {

		refName, pushErr := p.PrepareAndPushReleaseBranch(ctx, ctx.Param("release"))
		if pushErr != nil {
			ctx.JSON(500, gin.H{
				"error": pushErr.Error(),
			})
			return
		}

		prID, createPR := p.CreatePR(ctx, refName)
		if createPR != nil {
			ctx.JSON(500, gin.H{
				"error": createPR.Error(),
			})
			return
		}

		ctx.JSON(200, gin.H{
			"prID": prID, // wk
		})
	})

	r.POST("/approve/:release/:prID", func(ctx *gin.Context) {
		mergeErrr := p.MergePR(ctx, github.PullRequestID(ctx.Param("prID")))
		if mergeErrr != nil {
			ctx.JSON(500, gin.H{
				"error": mergeErrr.Error(),
			})
			return
		}

		deleteBranchErr := p.DeleteReleaseBranch(ctx, github.RefName(ctx.Param("release")))
		if deleteBranchErr != nil {
			ctx.JSON(500, gin.H{
				"error": deleteBranchErr.Error(),
			})
			return
		}

		ctx.JSON(204, nil)
	})

	return r.Run(":9090")
}
