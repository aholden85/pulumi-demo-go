//go:build integration
// +build integration

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/pulumi/pulumi/pkg/v3/testing/integration"

	"github.com/stretchr/testify/assert"

	"gopkg.in/yaml.v3"
)

// TODO: Move the pulumiConfig-related vars/funcs to a separate file/module.
const pulumiConfigFileName = "Pulumi.dev.yaml"
const errorMessageTemplate = "\n[%s]\t(integration_test.go::%s): %s"

var printColors = map[string]string{
	"Reset":  "\033[0m",
	"Red":    "\033[31m",
	"Green":  "\033[32m",
	"Yellow": "\033[33m",
	"Blue":   "\033[34m",
	"Purple": "\033[35m",
	"Cyan":   "\033[36m",
	"White":  "\033[37m",
}

type PulumiConfig struct {
	Config map[string]string `yaml:"config"`
}

type Image struct {
	Url  string    `json:"url"`
	Tags ImageTags `json:"tags"`
}

type ImageTags map[string]string

func getPaths() []string {
	return []string{
		"images",
		"facts",
	}
}

// This currently isn't required as the test packages are part of the
// IaC module, where the Fact struct is already defined. We will need
// to uncomment this if/when that changes.
//
// type Fact struct {
// 	FactId	int 	`dynamodbav:"FactId" json:"id"`
// 	Text	string	`dynamodbav:"Text" json:"text"`
// }

func stringInArray(str string, arr []string) bool {
	for _, elem := range arr {
		if elem == str {
			return true
		}
	}
	return false
}

func printLogMsg(errorType string, functionName string, message string) {
	fmt.Printf(
		errorMessageTemplate,
		errorType,
		functionName,
		message,
	)
}

func getPathStruct(path string) interface{} {
	switch path {
	case "images":
		return Image{}
	case "facts":
		return Fact{}
	default:
		printLogMsg(
			"Error",
			"getPathStruct",
			fmt.Sprintf(
				"No struct defined for path '%s'",
				path,
			),
		)
		return nil
	}
}

func copyAssets() {
	// Copy the asset folder so we can test their integration
	// This is done because the assets are contained in a different directory
	// to the IaC code:
	//
	//	pulumi-demo-go
	//	├─  assets
	//	└─  iac
	//
	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		printLogMsg(
			"Error",
			"copyAssets",
			fmt.Sprintf(
				"Unable to get current working directory with 'os.Getwd()'! '%s'",
				err,
			),
		)
	}

	liveAssetFolderPath := path.Join(currentWorkingDirectory, "..", "assets")
	testAssetFolderPath := path.Join(os.Getenv("HOME"), "go/src/assets")

	copyCmd := exec.Command("cp", "-R", liveAssetFolderPath, testAssetFolderPath)
	_, err = copyCmd.Output()
	if err != nil {
		printLogMsg(
			"Error",
			"copyAssets",
			fmt.Sprintf(
				"Folder copy from '%s' to '%s' failed! '%s'",
				liveAssetFolderPath,
				testAssetFolderPath,
				err,
			),
		)
	}
}

func removeAssets() {
	testAssetFolderPath := path.Join(os.Getenv("HOME"), "go/src/assets")

	deleteCmd := exec.Command("rm", "-rf", testAssetFolderPath)
	deleteCmd.Run()
}

func getConfigVars() PulumiConfig {
	// Read the file
	data, err := os.ReadFile(pulumiConfigFileName)
	if err != nil {
		printLogMsg(
			"Error",
			"getConfigVars",
			fmt.Sprintf(
				"Unable to read YAML file `%s`! '%s'",
				pulumiConfigFileName,
				err,
			),
		)
	}

	// Create a struct to hold the YAML data
	var pConfig PulumiConfig

	// Unmarshal the YAML data into the struct
	err = yaml.Unmarshal(data, &pConfig)
	if err != nil {
		printLogMsg(
			"Error",
			"getConfigVars",
			fmt.Sprintf(
				"Failed to unmarshal YAML file `%s`! '%s'",
				pulumiConfigFileName,
				err,
			),
		)
	}

	return pConfig
}

func validateResourceCounts(t *testing.T, stack integration.RuntimeValidationStackInfo) {
	// TODO: Instead of counts, have URNs (map[string][]string)
	// Define a common value to be used for any dynamic counts.
	dynamicCountPlaceholder := -1
	expectedResourceCounts := map[string]int{
		"aws-apigateway:index:RestAPI":                           1,
		"aws:apigateway/deployment:Deployment":                   1,
		"aws:apigateway/restApi:RestApi":                         1,
		"aws:apigateway/stage:Stage":                             1,
		"aws:dynamodb/table:Table":                               2,
		"aws:dynamodb/tableItem:TableItem":                       dynamicCountPlaceholder,
		"aws:iam/role:Role":                                      3,
		"aws:iam/rolePolicy:RolePolicy":                          3,
		"aws:iam/rolePolicyAttachment:RolePolicyAttachment":      3,
		"aws:lambda/function:Function":                           3,
		"aws:lambda/permission:Permission":                       3,
		"aws:s3/bucket:Bucket":                                   1,
		"aws:s3/bucketObject:BucketObject":                       dynamicCountPlaceholder,
		"aws:s3/bucketPolicy:BucketPolicy":                       1,
		"aws:s3/bucketPublicAccessBlock:BucketPublicAccessBlock": 1,
		"pulumi:providers:aws-apigateway":                        1,
		"pulumi:providers:aws":                                   2,
		"pulumi:providers:pulumi":                                1,
		"pulumi:pulumi:Stack":                                    1,
	}

	var actualResourceCounts map[string]int = map[string]int{}

	for _, resource := range stack.Deployment.Resources {
		_, found := actualResourceCounts[string(resource.Type)]
		if !found {
			actualResourceCounts[string(resource.Type)] = 0
		}
		actualResourceCounts[string(resource.Type)]++
	}

	var issuesFound bool = false

	for key := range expectedResourceCounts {
		// For any counts that are dynamic, replace their count with the actual
		// count
		if expectedResourceCounts[key] == dynamicCountPlaceholder {
			expectedResourceCounts[key] = actualResourceCounts[key]
		}

		if expectedResourceCounts[key] != actualResourceCounts[key] {
			issuesFound = true
		}
	}

	// Check for any created resources that were not expected.
	for key := range actualResourceCounts {
		_, ok := expectedResourceCounts[key]
		if !ok {
			expectedResourceCounts[key] = 0
			issuesFound = true
		}
	}

	assert.False(
		t,
		issuesFound,
		"Difference between expected and actual resource count!",
	)

	if issuesFound {
		resourceCountSummary := "\n" +
			"\tYou will need to update the 'expectedResourceCounts' map in the\n" +
			"\t'integration_test.go' file. Please review the summary below and\n" +
			"\tthe differences between the expected/actual counts of each\n" +
			"\tresource:\n" +
			"\n" +
			fmt.Sprintf("+%s", strings.Repeat("-", 17)) +
			fmt.Sprintf("+%s+\n", strings.Repeat("-", 73)) +
			fmt.Sprintf("| Expect / Actual | Resource%s|\n", strings.Repeat(" ", 64)) +
			fmt.Sprintf("+%s", strings.Repeat("-", 17)) +
			fmt.Sprintf("+%s+\n", strings.Repeat("-", 73))

		for key := range expectedResourceCounts {
			if expectedResourceCounts[key] == 0 {
				resourceCountSummary += printColors["Red"]
			} else if expectedResourceCounts[key] != actualResourceCounts[key] {
				resourceCountSummary += printColors["Yellow"]
			} else {
				resourceCountSummary += printColors["Green"]
			}
			resourceCountSummary += fmt.Sprintf(
				"| %6d / %6d | %-71s |\n",
				expectedResourceCounts[key],
				actualResourceCounts[key],
				key,
			)
		}

		resourceCountSummary += fmt.Sprintf(
			"%s\n\n",
			printColors["Reset"]+
				fmt.Sprintf("+%s", strings.Repeat("-", 17))+
				fmt.Sprintf("+%s+\n", strings.Repeat("-", 73)),
		)

		printLogMsg(
			"Error",
			"validateResourceCounts",
			resourceCountSummary,
		)
	}
}

func runtimeValidation(t *testing.T, stack integration.RuntimeValidationStackInfo) {
	fmt.Printf("\tCOMPLETE\n")
	fmt.Printf("\tValidating that the expected number of resources will be created...")
	validateResourceCounts(t, stack)
	fmt.Printf("\tCOMPLETE\n")
}

func TestIntegration(t *testing.T) {
	fmt.Printf("Executing ~INTEGRATION~ tests...\n")
	// Copy the asset folder so we can test their integration
	fmt.Printf("\tCopying asset folder and contents...")
	copyAssets()
	fmt.Printf("\tCOMPLETE\n")

	currentWorkingDirectory, _ := os.Getwd()

	fmt.Printf("\tVerifying that the Pulumi code compiles correctly...")

	integration.ProgramTest(t, &integration.ProgramTestOptions{
		Quick:                  true,
		SkipRefresh:            true,
		Dir:                    currentWorkingDirectory,
		Config:                 getConfigVars().Config,
		ExtraRuntimeValidation: runtimeValidation,
	})

	// Clean up the copied asset folder
	fmt.Printf("\tRemoving copied asset folder and contents...")
	removeAssets()
	fmt.Printf("\tCOMPLETE\n")
}
