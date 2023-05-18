package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const acronymEnvVar = string("ACRONYM")
const acronymDefault = string("xaas")
const patFormat = string("%s_pat_%s")
const patSuffixLength = int(64)
const tableNameEnvVar = string("PAT_TABLE_NAME")
const tableNameDefault = string("xaas-api-pats")

var ddbClient dynamodb.Client
var acronym string
var tableName string
var sequenceLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

type Pat struct {
	Pat string `dynamodbav:"Pat" json:"pat"`
}

func init() {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ddbClient = *dynamodb.NewFromConfig(sdkConfig)

	// Grab the acronym from the environment variables.
	// If the environment variable is not defined, fall back to a default.
	acronym = os.Getenv(acronymEnvVar)
	if len(acronym) == 0 {
		acronym = acronymDefault
	}

	// Grab the DynamoDB table name from the environment variables.
	// If the environment variable is not defined, fall back to a default.
	tableName = os.Getenv(tableNameEnvVar)
	if len(tableName) == 0 {
		tableName = tableNameDefault
	}
}

func clientError(status int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

func isDuplicatePat(ctx context.Context, pat string) (bool, error) {
	// Grab the name of the pat table from the environment variables.
	// If the environment variable is not defined, fall back to a default.
	tableKey, err := attributevalue.Marshal(pat)
	if err != nil {
		return false, err
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"Pat": tableKey,
		},
	}

	result, err := ddbClient.GetItem(ctx, input)
	if err != nil {
		return false, err
	}

	if result.Item != nil {
		log.Printf("PAT '%s' is a duplicate", pat)
		return true, nil
	} else {
		log.Printf("PAT '%s' is NOT a duplicate", pat)
		return false, nil
	}
}

func generatePat() string {
	patSuffix := make([]rune, patSuffixLength)
	for i := range patSuffix {
		patSuffix[i] = sequenceLetters[rand.Intn(len(sequenceLetters))]
	}

	return fmt.Sprintf(patFormat, acronym, string(patSuffix))
}

func generateUniquePat(ctx context.Context) (string, error) {
	pat := string("")

	for {
		pat = generatePat()
		isDuplicate, err := isDuplicatePat(ctx, pat)
		if err != nil {
			return string(""), err
		}
		if !isDuplicate {
			break
		}
	}

	return pat, nil
}

func postNewPat(ctx context.Context) (*Pat, error) {
	pat, err := generateUniquePat(ctx)
	if err != nil {
		return nil, err
	}

	patStruct := Pat{
		Pat: pat,
	}

	item, err := attributevalue.MarshalMap(patStruct)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	}

	_, err = ddbClient.PutItem(ctx, input)
	if err != nil {
		return nil, err
	}

	return &patStruct, nil
}

func deletePat(ctx context.Context, pat string) error {
	patStruct := Pat{
		Pat: pat,
	}

	item, err := attributevalue.MarshalMap(patStruct)
	if err != nil {
		return err
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key:       item,
	}

	_, err = ddbClient.DeleteItem(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

func processPost(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	pat, err := postNewPat(ctx)
	if err != nil {
		log.Printf("Failed to create new pat: %s", err)
		return serverError(err)
	}

	if pat == nil {
		log.Printf("nil returned from postNewPat!")
		return clientError(http.StatusInternalServerError)
	}

	json, err := json.Marshal(pat)
	if err != nil {
		log.Printf("Failed to json.Marshal(pat): %s", err)
		return serverError(err)
	}
	log.Printf("Successfully create new pat: %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func processDelete(ctx context.Context, pat string) (events.APIGatewayProxyResponse, error) {
	err := deletePat(ctx, pat)
	if err != nil {
		log.Printf("Failed to delete pat: %s", err)
		return serverError(err)
	}

	json, err := json.Marshal(pat)
	if err != nil {
		log.Printf("Failed to json.Marshal(pat): %s", err)
		return serverError(err)
	}
	log.Printf("Successfully deleted pat: %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "POST":
		return processPost(ctx)
	case "DELETE":
		suppliedPat, ok := req.Headers["Authorization"]
		if !ok {
			return clientError(http.StatusUnauthorized)
		}
		return processDelete(ctx, suppliedPat)
	default:
		return clientError(http.StatusMethodNotAllowed)
	}
}

func main() {
	lambda.Start(router)
}
