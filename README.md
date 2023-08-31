Ecosystem QE System Testing Automation
=======
# eco-gosystem

## Overview
The [eco-gosystem](https://github.com/openshift-kni/eco-gosystem) project provides system level tests which can run against various OpenShift clusters.
The project is based on golang+[ginkgo](https://onsi.github.io/ginkgo) framework and it makes use of packages from [eco-goinfra](https://github.com/openshift-kni/eco-goinfra)

### Project requirements
* golang v1.19.x
* ginkgo v2.x

## eco-gosystem
The  [eco-gosystem](https://github.com/openshift-kni/eco-gosystem) is designed to test a pre-installed OCP cluster which meets the following requirements:

### Mandatory setup requirements:
* OCP cluster installed with version >=4.10

#### Optional:
* PTP operator
* SRIOV operator
* SRIOV-FEC operator
* Local Storage operator
* Cluster Logging operator
* RAN DU profile

### Supported setups:
* Regular 3 master nodes with 2 or more workers clusters
* Compact 3 master node clusters
* Single node clusters

**WARNING!**: Depending on the test configuration some of the tests may require access to physical resources such as network physical functions or accelerators.

### General environment variables
#### Mandatory:
* `KUBECONFIG` - Path to kubeconfig file. Default: empty
#### Optional:
* Logging with glog

We use glog library for logging in the project. In order to enable verbose logging the following needs to be done:

1. Make sure to import inittool in your go script, per this example:

<sup>
    import (
      . "github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
    )
</sup>

2. Need to export the following SHELL variable:
> export ECO_VERBOSE_LEVEL=100

##### Notes:

  1. The value for the variable has to be >= 100.
  2. The variable can simply be exported in the shell where you run your automation.
  3. The go file you work on has to be in a directory under github.com/openshift-kni/eco-gosystem/tests/ directory for being able to import inittools.
  4. Importing inittool also intializes the apiclient and it's available via "APIClient" variable.

* Collect logs from cluster with reporter

We use k8reporter library for collecting resource from cluster in case of test failure.
In order to enable k8reporter the following needs to be done:

1. Export DUMP_FAILED_TESTS and set it to true. Use example below
> export ECO_DUMP_FAILED_TESTS=true

2. Specify absolute path for logs directory like it appears below. By default /tmp/reports directory is used.
> export ECO_REPORTS_DUMP_DIR=/tmp/logs_directory

* Generation Polarion XML reports

We use polarion library for generating polarion compatible xml reports. 
The reporter is enabled by default and stores reports under REPORTS_DUMP_DIR directory.
In oder to disable polarion reporter the following needs to be done:
> export ECO_POLARION_REPORT=false


<!-- TODO Update this section with optional env vars for each test suite -->

## How to run

The test-runner [script](scripts/test-runner.sh) is the recommended way for executing tests.

Parameters for the script are controlled by the following environment variables:
- `ECO_TEST_FEATURES`: list of features to be tested ("all" will include all tests). All subdirectories under tests that match a feature will be included (internal directories are excluded) - _required_
- `ECO_TEST_LABELS`: ginkgo query passed to the label-filter option for including/excluding tests - _optional_ 
- `ECO_VERBOSE_SCRIPT`: prints verbose script information when executing the script - _optional_
- `ECO_TEST_VERBOSE`: executes ginkgo with verbose test output - _optional_
- `ECO_TEST_TRACE`: includes full stack trace from ginkgo tests when a failure occurs - _optional_

It is recommended to execute the runner script through the `make run-tests` make target.

Example:
```
$ export KUBECONFIG=/path/to/kubeconfig
$ export ECO_TEST_FEATURES="ran-du"
$ export ECO_TEST_LABELS='launch-workload'
$ make run-tests
Executing eco-gosystem test-runner script
scripts/test-runner.sh
ginkgo -timeout=24h --keep-going --require-suite -r --label-filter="launch-workload" ./tests/ran-du
```
# eco-gosystem - How to contribute

The project uses a development method - forking workflow
### The following is a step-by-step example of forking workflow:
1) A developer [forks](https://docs.gitlab.com/ee/user/project/repository/forking_workflow.html#creating-a-fork)
   the [eco-gosystem](https://github.com/openshift-kni/eco-gosystem) project
2) A new local feature branch is created
3) A developer makes changes on the new branch.
4) New commits are created for the changes.
5) The branch gets pushed to developer's own server-side copy.
6) Changes are tested.
7) A developer opens a pull request(`PR`) from the new branch to
   the [eco-gosystem](https://github.com/openshift-kni/eco-gosystem).
8) The pull request gets approved from at least 2 reviewers for merge and is merged into
   the [eco-gosystem](https://github.com/openshift-kni/eco-gosystem) .

# eco-gosystem - Project structure

    ├── scripts                                  # makefile scripts
    ├── tests                                    # test cases directory
    │   ├── internal                             # common packages used acrossed framework
    │   │   ├── await                            # package providing function which wait for certain conditions
    │   │   ├── config                           # common config struct used across framework
    │   │   ├── inittools                        # package used for initializing api client and loading configurations
    │   │   ├── params                           # common constant and parameters used acrossed framework
    │   │   └── shell                            # package providing shell commands execution
    │   └── ran-du                               # directory hosting system level tests specific to RAN DU use case
    │       ├── internal                         # internal packages within the ran-du test suite
    │       ├── ran_du_suite_test.go             # ran-du test suite file
    │       └── tests                            # ran-du tests directory
    │           └── launch-workload.go           # ran-du launch workload test
    └── vendor                                   # dependencies folder

### Code conventions
#### Lint
Push requested are tested in a pipeline with golangci-lint. It is advised to add [Golangci-lint integration](https://golangci-lint.run/usage/integrations/) to your development editor. It's recommended to run `make lint` before uploading a PR.

#### Commit Message Guidelines
There are two main components of a Git commit message: the title or summary, and the description. The commit message title is limited to 72 characters, and the description has no character limit.

Commit title should be a brief description of the change: Example - "added deployment test". The commit description should provide a more detailed description of the change.

#### Functions format
If the function's arguments fit in a single line - use the following format:
```go
func Function(argInt1, argInt2 int, argString1, argString2 string) {
    ...
}
```

If the function's arguments don't fit in a single line - use the following format:
```go
func Function(
    argInt1 int,
    argInt2 int,
    argInt3 int,
    argInt4 int,
    argString1 string,
    argString2 string,
    argString3 string,
    argString4 string) {
    ...
}
```
One more acceptable format example:
```go
func Function(
    argInt1, argInt2 int, argString1, argString2 string, argSlice1, argSlice2 []string) {

}
```

### Common issues:
* If the automated commit check fails - make sure to pull/rebase the latest change and have a successful execution of 'make lint' locally first.
