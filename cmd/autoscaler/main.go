package main

import autoscaler "github.com/Kenneth-KT/k8s-rds-autoscaler/pkg/rds-autoscaler"

func main() {
	autoscaler.RunController()
}
