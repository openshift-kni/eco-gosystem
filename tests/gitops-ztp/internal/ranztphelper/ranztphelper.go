package ranztphelper

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gosystem/tests/gitops-ztp/internal/params"
)

var (
	HubAPIClient   *clients.Settings
	HubName        string
	SpokeAPIClient *clients.Settings
	SpokeName      string
	ArgocdApps     = map[string]params.ArgocdGitDetails{}
	ZtpVersion     string
	// AcmVersion        string
	// TalmVersion       string
	// pidAndAffinityExp = regexp.MustCompile(`pid (\d+)'s current affinity list: (.*)$`)
)

// GetZtpContext is used to get the context for the Ztp test client interactions.
func GetZtpContext() context.Context {
	return context.Background()
}

// GetAllTestClients is used to quickly obtain a list of all the test clients.
func GetAllTestClients() []*clients.Settings {
	return []*clients.Settings{
		HubAPIClient,
		SpokeAPIClient,
	}
}

// IsVersionStringInRange can be used to check if a version string is between a specified min and max value.
// All the string inputs to this function should be dot separated positive intergers, e.g. "1.0.0" or "4.10".
// Each string inputs must be at least two dot separarted integers but may also be 3 or more.
func IsVersionStringInRange(version, minimum, maximum string) bool {
	// First we will define a helper to validate that a string is at least two dot separated positive integers
	ValidateInputString := func(input string) (bool, []int) {
		// Split the input string on the dot character
		versionSplits := strings.Split(input, ".")

		// We need at least two digits to verify
		if len(versionSplits) < 2 {
			return false, []int{}
		}

		// Prepare a list of digits to return later
		digits := []int{}

		// Check if the first two splits are valid integers
		for i := 0; i < 2; i++ {
			// Attempt to convert the string to an integer
			digit, err := strconv.Atoi(versionSplits[i])
			if err != nil {
				return false, []int{}
			}

			// Save the digit
			digits = append(digits, digit)
		}

		// Otherwise if we've passed all the checks it is a valid version string
		return true, digits
	}

	versionValid, versionDigits := ValidateInputString(version)
	minimumValid, minimumDigits := ValidateInputString(minimum)
	maximumValid, maximumDigits := ValidateInputString(maximum)

	if !minimumValid {
		// We don't want any bad minimums to be passed at all, so panic if minimum wasn't an empty string
		if minimum != "" {
			panic(fmt.Errorf("invalid minimum provided: '%s'", minimum))
		}

		// Assume the minimum digits are [0,0] for later comparison
		minimumDigits = []int{0, 0}
	}

	if !maximumValid {
		// We don't want any bad maximums to be passed at all, so panic if maximum wasn't an empty string
		if maximum != "" {
			panic(fmt.Errorf("invalid maximum provided: '%s'", maximum))
		}

		// Assume the maximum digits are [math.MaxInt, math.MaxInt] for later comparison
		maximumDigits = []int{math.MaxInt, math.MaxInt}
	}

	// If the version was not valid then we need to check the min and max
	if !versionValid {
		// If no min or max was defined then return true
		if !minimumValid && !maximumValid {
			return true
		}

		// Otherwise return whether the input maximum was an empty string or not
		return maximum == ""
	}

	// Otherwise the versions were valid so compare the digits
	for i := 0; i < 2; i++ {
		// The version bit should be between the minimum and maximum
		if versionDigits[i] < minimumDigits[i] || versionDigits[i] > maximumDigits[i] {
			return false
		}
	}

	// At the end if we never returned then all the digits were in valid range
	return true
}

// JoinGitPaths is used to join any combination of git strings but also avoiding double slashes.
func JoinGitPaths(inputs []string) string {
	// We want to preserve any existing double slashes but we don't want to add any between the input elements
	// To work around this we will use a special join character and a couple replacements
	special := "<<join>>"

	// Combine the inputs with the special join character
	result := strings.Join(
		inputs,
		special,
	)

	// Replace any special joins that have a slash prefix
	result = strings.ReplaceAll(result, "/"+special, "/")

	// Replace any special joins that have a slash suffix
	result = strings.ReplaceAll(result, special+"/", "/")

	// Finally replace any remaining special joins
	result = strings.ReplaceAll(result, special, "/")

	// The final result should never have double slashes between the joined elements
	// However if they already had any double slashes, e.g. "http://", they will be preserved
	return result
}

// DoesGitPathExist checks if the specified git url exists.
func DoesGitPathExist(gitURL, gitBranch, gitPath string) bool {
	// Combine the separate pieces to get the url
	// We also need to remove the ".git" from the end of the url
	url := JoinGitPaths(
		[]string{
			strings.Replace(gitURL, ".git", "", 1),
			"raw",
			gitBranch,
			gitPath,
		},
	)

	// Log the url we are trying
	log.Printf("Checking if git url '%s' exists\n", url)

	// Create a custom http.Transport with insecure TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Create a custom http.Client using the insecure transport
	client := &http.Client{Transport: transport}

	// Make a request using the custom client
	resp, err := client.Get(url)

	// Check if we got a valid response
	if err == nil && resp.StatusCode == 200 {
		log.Printf("found valid git url for '%s'\n", gitPath)

		return true
	}

	// If we got here then none of the urls could be found.
	log.Printf("could not find valid url for '%s'\n", gitPath)

	return false
}
