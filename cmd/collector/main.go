package main

import (
	"fmt"
	"os"

	"github.com/kwiesmueller/k8s-status-metrics/pkg/collector"
)

func main() {
	app := collector.NewCLIApp()
	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
