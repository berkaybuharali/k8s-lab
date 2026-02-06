// cli/main.go
// Package main is the entry point for the k8s-lab CLI tool.
// This CLI manages Kubernetes lab environments across cloud providers
// with infrastructure provisioning, cluster bootstrapping, and backup/restore.
package main

import "github.com/berkaybuharali/k8s-lab/cli/cmd"

func main() {
	cmd.Execute()
}
