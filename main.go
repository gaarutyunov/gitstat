package main

import (
	"context"
	"github.com/gaarutyunov/gitstat/cli"
	"github.com/sirupsen/logrus"
)

func main() {
	err := cli.Execute(context.Background())
	if err != nil {
		logrus.Fatal(err)
	}
}
