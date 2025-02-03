package main

import (
	"os"

	"github.com/glothriel/temporaldemo/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	if startErr := cmd.Start(os.Args); startErr != nil {
		logrus.Fatal(startErr)
	}
}
