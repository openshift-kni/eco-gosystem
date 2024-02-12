package randuparams

import (
	"time"
)

const (
	// Label represents RAN DU system tests label that can be used for test cases selection.
	Label = "randu"
	// LabelLaunchWorkloadTestCases represents tests labels related to test workload.
	LabelLaunchWorkloadTestCases = "launch-workload"
	// DefaultTimeout is the timeout used for test resources creation.
	DefaultTimeout = 300 * time.Second
	// TestWorkloadShellLaunchMethod is used when usin a shell script for launching the test workload.
	TestWorkloadShellLaunchMethod = "shell"
	// DU Hosts kubeconfig env var.
	KubeEnvKey string = "KUBECONFIG"
)
