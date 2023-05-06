package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	// "reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"github.com/pulumi/pulumi-aws-apigateway/sdk/go/apigateway"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
)

var parentFolderPath string
var assetFolderPath string
var animalName string
var acronym string
var animalAssetFolderPath string
var animalImageFolderPath string
var imageMetadataFile string
var imageMetadataPath string
var factFile string
var lambdaFolder string
var lambdaZipSuffix string
var createdInfrastructure Infrastructure

// TODO: Break this down into several types
type MetadataImageList map[string]map[string]map[string]string

type Fact struct {
	FactId int    `dynamodbav:"FactId" json:"id"`
	Text   string `dynamodbav:"Text" json:"text"`
}

func (fact Fact) MarshalToDynamoDB() string {
	return fmt.Sprintf(
		"{\n  \"FactId\": {\"N\": \"%d\"},\n  \"Text\": {\"S\": \"%s\"}\n}\n",
		fact.FactId,
		fact.Text,
	)
}

type Infrastructure struct {
	DdbTableItems []*dynamodb.TableItem
	DdbTables     []*dynamodb.Table
	Lambdas       []*lambda.Function
	RestApis      []*apigateway.RestAPI
	S3Buckets     []*s3.Bucket
	S3Objects     []*s3.BucketObject
}

type LambdaInfra struct {
	Name   string
	Lambda *lambda.Function
	Role   *iam.Role
	Routes []apigateway.RouteArgs
}

type LambdaRoute struct {
	Path   string
	Method apigateway.Method
}

type RolePolicy struct {
	NameSuffix string
	Document   pulumi.StringOutput
}

func initStrings(ctx *pulumi.Context) {
	cwd, _ := os.Getwd()
	parentFolderPath = path.Join(cwd, "..") + "/"
	assetFolderPath = path.Join(cwd, "..", "assets")
	conf := config.New(ctx, "")
	animalName = conf.Require("animal")
	acronym = fmt.Sprintf("%caas", animalName[0])
	animalAssetFolderPath = fmt.Sprintf("%s/animals/%s", assetFolderPath, animalName)
	animalImageFolderPath = fmt.Sprintf("%s/images/", animalAssetFolderPath)
	imageMetadataFile = "metadata.json"
	imageMetadataPath = fmt.Sprintf("%s/%s", animalImageFolderPath, imageMetadataFile)
	factFile = fmt.Sprintf("%s/facts.txt", animalAssetFolderPath)
	lambdaFolder = fmt.Sprintf("%s/lambda", assetFolderPath)
	lambdaZipSuffix = "bin/main.zip"
}

// As we can't declare const arrays, we use the functions below.
func getLambdaNames() []string {
	return []string{
		"facts",
		"images",
	}
}

// func getCurrentAccountId(ctx *pulumi.Context) (string, error) {
// 	// Get the AWS Account ID that we're deploying to
// 	currentCaller, err := aws.GetCallerIdentity(ctx, nil, nil)
// 	if err != nil {
// 		return "", err
// 	}
// 	return currentCaller.AccountId, nil
// }

// func getCurrentRegion(ctx *pulumi.Context) (string, error) {
// 	// Get the AWS region that we're deploying to
// 	currentRegion, err := aws.GetRegion(ctx, nil, nil)
// 	if err != nil {
// 		return "", err
// 	}
// 	return currentRegion.Name, nil
// }

// deployPublicBucket creates an s3.Bucket object, applies a permissive
// PublicAccessBlock, and a public BucketPolicy.
func deployPublicBucket(ctx *pulumi.Context, bucketName string) (*s3.Bucket, error) {
	// Create an AWS resource (S3 Bucket)
	bucket, err := s3.NewBucket(
		ctx,
		bucketName,
		&s3.BucketArgs{},
	)
	if err != nil {
		return nil, err
	}

	// Add the resource to createdInfrastructure for testing purposes.
	createdInfrastructure.S3Buckets = append(
		createdInfrastructure.S3Buckets,
		bucket,
	)

	// Create an open Public Access Block
	publicAccessBlock, err := s3.NewBucketPublicAccessBlock(
		ctx,
		fmt.Sprintf("%s-publicaccess-allow", bucketName),
		&s3.BucketPublicAccessBlockArgs{
			Bucket:                bucket.ID(),
			BlockPublicAcls:       pulumi.Bool(false),
			BlockPublicPolicy:     pulumi.Bool(false),
			IgnorePublicAcls:      pulumi.Bool(false),
			RestrictPublicBuckets: pulumi.Bool(false),
		},
	)
	if err != nil {
		return nil, err
	}

	// Create a public read policy for the S3 bucket
	_, err = s3.NewBucketPolicy(
		ctx,
		fmt.Sprintf("%s-assets-policy", acronym),
		&s3.BucketPolicyArgs{
			Bucket: bucket.ID(),
			Policy: pulumi.Any(map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Effect":    "Allow",
						"Principal": "*",
						"Action": []interface{}{
							"s3:GetObject",
						},
						"Resource": []interface{}{
							pulumi.Sprintf("arn:aws:s3:::%s/*", bucket.ID()),
						},
					},
				},
			}),
		},
		pulumi.DependsOn([]pulumi.Resource{
			publicAccessBlock,
		}),
	)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

func compileLambda(module string) error {
	// Build and zip the code
	cmd := exec.Command("make", "build")
	cmd.Dir = fmt.Sprintf("%s/%s", lambdaFolder, module)
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

func deployLambdaFunction(
	// Arguments
	ctx *pulumi.Context,
	lambdaName string,
	rolePolicies []RolePolicy,
	envVars pulumi.StringMap,
	routes []LambdaRoute,
) (
	// Return objects
	LambdaInfra,
	error,
) {
	resourceNamePrefix := fmt.Sprintf("%s-lambda-%s", acronym, lambdaName)
	// Deploy the IAM role for the Lambda function
	role, err := iam.NewRole(
		ctx,
		fmt.Sprintf("%s-exec-role", resourceNamePrefix),
		&iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(
				`{
					"Version": "2012-10-17",
					"Statement": [{
						"Sid": "",
						"Effect": "Allow",
						"Principal": {
							"Service": "lambda.amazonaws.com"
						},
						"Action": "sts:AssumeRole"
					}]
				}`,
			),
		},
	)
	if err != nil {
		return LambdaInfra{}, err
	}

	// Attach the AWSLambdaBasicExecutionRole policy to allow the Lambda
	// functions to write to CloudWatch
	_, err = iam.NewRolePolicyAttachment(
		ctx,
		fmt.Sprintf("%s-exec-role-cwpolicy", resourceNamePrefix),
		&iam.RolePolicyAttachmentArgs{
			Role: role,
			PolicyArn: pulumi.String(
				"arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
			),
		},
	)
	if err != nil {
		return LambdaInfra{}, err
	}

	// Compile the Lambda code
	err = compileLambda(lambdaName)
	if err != nil {
		return LambdaInfra{}, err
	}

	var policies []pulumi.Resource

	for _, rolePolicy := range rolePolicies {
		// Attach IAM policies to the IAM role
		thisPolicy, err := iam.NewRolePolicy(
			ctx,
			fmt.Sprintf(
				"%s-%s",
				resourceNamePrefix,
				rolePolicy.NameSuffix,
			),
			&iam.RolePolicyArgs{
				Role:   role.Name,
				Policy: rolePolicy.Document,
			},
		)
		if err != nil {
			return LambdaInfra{}, err
		}
		policies = append(policies, thisPolicy)
	}

	// Create the Lambda function
	function, err := lambda.NewFunction(
		ctx,
		resourceNamePrefix,
		&lambda.FunctionArgs{
			Handler: pulumi.String("main"),
			Role:    role.Arn,
			Runtime: pulumi.String("go1.x"),
			Code: pulumi.NewFileArchive(
				fmt.Sprintf("%s/%s/%s", lambdaFolder, lambdaName, lambdaZipSuffix),
			),
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: envVars,
			},
		},
		pulumi.DependsOn(policies),
	)
	if err != nil {
		return LambdaInfra{}, err
	}

	// Add the resource to createdInfrastructure for testing purposes.
	createdInfrastructure.Lambdas = append(
		createdInfrastructure.Lambdas,
		function,
	)

	apiGwRoutes := make([]apigateway.RouteArgs, 0)
	for _, route := range routes {
		apiGwRoutes = append(
			apiGwRoutes,
			apigateway.RouteArgs{
				Path:         route.Path,
				Method:       &route.Method,
				EventHandler: function,
			},
		)
	}

	infra := LambdaInfra{
		Name:   lambdaName,
		Lambda: function,
		Role:   role,
		Routes: apiGwRoutes,
	}
	return infra, nil
}

func deployLambdaStack(ctx *pulumi.Context, lambdaName string) (LambdaInfra, error) {
	switch lambdaName {
	case "images":
		bucket, err := deployPublicBucket(
			ctx,
			fmt.Sprintf("%s-s3-assets", acronym),
		)
		if err != nil {
			return LambdaInfra{}, err
		}

		// Deploy the images in the image folder to the S3 bucket
		err = addFolderContentsToS3(ctx, animalImageFolderPath, bucket)
		if err != nil {
			return LambdaInfra{}, err
		}

		// Create a list of IAM policies required by the "images" lambda.
		// Specifically, we need to be able to read
		policies := []RolePolicy{
			{
				NameSuffix: "s3-read-policy",
				Document: pulumi.Sprintf(
					`{
						"Version": "2012-10-17",
						"Statement": [
							{
								"Sid": "DescribeImagesBucket",
								"Effect": "Allow",
								"Action": [
									"s3:GetBucketLocation",
									"s3:GetObject",
									"s3:GetObjectTagging",
									"s3:ListBucket"
								],
								"Resource": [
									"%s/*",
									"%s"
								]
							}
						]
					}`,
					bucket.Arn,
					bucket.Arn,
				),
			},
		}

		functionInfra, err := deployLambdaFunction(
			ctx,
			"images",
			policies,
			pulumi.StringMap{
				"IMAGES_BUCKET_NAME": bucket.Bucket,
				"IMAGES_OBJECT_PREFIX": pulumi.String(
					strings.TrimPrefix(
						animalImageFolderPath,
						parentFolderPath,
					),
				),
			},
			[]LambdaRoute{
				{
					Path:   "/images",
					Method: apigateway.MethodGET,
				},
			},
		)
		if err != nil {
			return LambdaInfra{}, err
		}

		return functionInfra, nil
	case "facts":
		// Create a DynamoDB table
		ddbTable, err := dynamodb.NewTable(
			ctx,
			fmt.Sprintf("%s-ddb-facts", acronym),
			&dynamodb.TableArgs{
				Attributes: dynamodb.TableAttributeArray{
					&dynamodb.TableAttributeArgs{
						Name: pulumi.String("FactId"),
						Type: pulumi.String("N"),
					},
				},
				HashKey:       pulumi.String("FactId"),
				BillingMode:   pulumi.String("PROVISIONED"),
				ReadCapacity:  pulumi.Int(10),
				WriteCapacity: pulumi.Int(10),
			},
		)
		if err != nil {
			return LambdaInfra{}, err
		}

		// Add the resource to createdInfrastructure for testing purposes.
		createdInfrastructure.DdbTables = append(
			createdInfrastructure.DdbTables,
			ddbTable,
		)

		// Deploy the facts to the DynamoDB table
		err = addTextContentsToDdb(ctx, factFile, ddbTable)
		if err != nil {
			return LambdaInfra{}, err
		}

		policies := []RolePolicy{
			{
				NameSuffix: "ddb-read-policy",
				Document: pulumi.Sprintf(
					`{
						"Version": "2012-10-17",
						"Statement": [
							{
								"Sid": "DescribeQueryScanFactsTable",
								"Effect": "Allow",
								"Action": [
									"dynamodb:DescribeTable",
									"dynamodb:Query",
									"dynamodb:Scan",
									"dynamodb:GetItem"
								],
								"Resource": "%s"
							}
						]
					}`,
					ddbTable.Arn,
				),
			},
		}

		functionInfra, err := deployLambdaFunction(
			ctx,
			"facts",
			policies,
			pulumi.StringMap{
				"FACTS_TABLE_NAME": ddbTable.Name,
			},
			[]LambdaRoute{
				{
					Path:   "/facts",
					Method: apigateway.MethodGET,
				},
			},
		)
		if err != nil {
			return LambdaInfra{}, err
		}

		return functionInfra, nil
	default:
		return LambdaInfra{}, fmt.Errorf(
			"undefined lambdaName passed to deployLambdaStack(): '%s'",
			lambdaName,
		)
	}
}

func addFolderContentsToS3(ctx *pulumi.Context, directory string, s3Bucket *s3.Bucket) error {
	// Get a list of all the files in the target folder
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return err
	}

	// Open the metadata file
	var metadata MetadataImageList
	metadataJson, err := os.Open(imageMetadataPath)
	if err != nil {
		return fmt.Errorf(
			"could not load the '%s' metadata file at: '%s'",
			animalName,
			imageMetadataPath,
		)
	} else {
		// Ensure that the metadata file is closed at the end of the function
		defer metadataJson.Close()

		byteValue, _ := ioutil.ReadAll(metadataJson)
		err = json.Unmarshal(byteValue, &metadata)
		if err != nil {
			return err
		}
	}

	for _, file := range files {
		// Skip the metadata file
		if file.Name() == imageMetadataFile {
			continue
		}

		objectTags := pulumi.ToStringMap(metadata["images"][file.Name()])

		bucketObject, err := s3.NewBucketObject(
			ctx,
			fmt.Sprintf("%s-s3-assets-%s", acronym, file.Name()),
			&s3.BucketObjectArgs{
				Bucket: s3Bucket,
				Key: pulumi.String(
					strings.TrimPrefix(
						animalImageFolderPath,
						parentFolderPath,
					) + file.Name(),
				),
				Source: pulumi.NewFileAsset(animalImageFolderPath + file.Name()),
				Tags:   objectTags,
			},
		)
		if err != nil {
			return err
		}

		// Add the resource to createdInfrastructure for testing purposes.
		createdInfrastructure.S3Objects = append(
			createdInfrastructure.S3Objects,
			bucketObject,
		)
	}

	return nil
}

func addTextContentsToDdb(ctx *pulumi.Context, filePath string, ddbTable *dynamodb.Table) error {
	// Open the text file
	textFile, err := os.Open(filePath)
	if err != nil {
		return err
	}

	// Ensure that the file is closed at the end of the function
	defer textFile.Close()

	// Create a scanner to read in the file line-by-line
	scanner := bufio.NewScanner(textFile)

	// Init a counter for the fact ID field.
	factId := 0

	//
	for scanner.Scan() {
		fact := Fact{
			FactId: factId,
			Text:   scanner.Text(),
		}

		tableItem, err := dynamodb.NewTableItem(
			ctx,
			fmt.Sprintf("%s-ddb-facts-%d", acronym, factId),
			&dynamodb.TableItemArgs{
				TableName: ddbTable.Name,
				HashKey:   ddbTable.HashKey,
				Item:      pulumi.String(fact.MarshalToDynamoDB()),
			},
		)
		if err != nil {
			return err
		}
		// Add the resource to createdInfrastructure for testing purposes.
		createdInfrastructure.DdbTableItems = append(
			createdInfrastructure.DdbTableItems,
			tableItem,
		)
		factId++
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func createInfrastructure(ctx *pulumi.Context) (*Infrastructure, error) {
	// Initialise paths and naming strings
	initStrings(ctx)

	// Create each of the Lambda functions and required resources
	lambdaFunctions := make([]LambdaInfra, 0)
	for _, lambdaName := range getLambdaNames() {
		// Deploy the Lambda function and retrieve required details
		functionInfra, err := deployLambdaStack(ctx, lambdaName)
		if err != nil {
			return nil, err
		}
		lambdaFunctions = append(lambdaFunctions, functionInfra)
	}

	// Collate the routes for each of the Lambda functions
	apiGatewayRoutes := make([]apigateway.RouteArgs, 0)
	for _, lambdaFunction := range lambdaFunctions {
		apiGatewayRoutes = append(apiGatewayRoutes, lambdaFunction.Routes...)
	}

	// Create the API Gateway resource to route requests to the Lambda
	// functions depending on defined paths
	api, err := apigateway.NewRestAPI(
		ctx,
		fmt.Sprintf("%s-apigw", acronym),
		&apigateway.RestAPIArgs{
			Routes: apiGatewayRoutes,
		},
	)
	if err != nil {
		return nil, err
	}

	// Add the resource to createdInfrastructure for testing purposes.
	createdInfrastructure.RestApis = append(
		createdInfrastructure.RestApis,
		api,
	)

	// The URL at which the REST API will be served
	ctx.Export("url", api.Url)

	return &createdInfrastructure, nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		createInfrastructure(ctx)
		return nil
	})
}
