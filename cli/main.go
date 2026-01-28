// cli/main.go
// Package main is the entry point for the k8s-lab CLI tool.
// This CLI manages Kubernetes lab environments across cloud providers
// with infrastructure provisioning, cluster bootstrapping, and backup/restore.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("k8s-lab CLI v0.1.0")
	os.Exit(0)
}
