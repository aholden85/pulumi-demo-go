package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
    "fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const tableNameEnvVar = "FACTS_TABLE_NAME"
const tableNameDefault = "xaas-api-facts"

var ddbClient dynamodb.Client

type Fact struct {
	FactId	int 	`dynamodbav:"FactId" json:"id"`
	Text	string	`dynamodbav:"Text" json:"text"`
}

func init() {
    sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ddbClient = *dynamodb.NewFromConfig(sdkConfig)
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

func getFact(ctx context.Context, factId int) (*Fact, error) {
    // Grab the name of the fact table from the environment variables.
    // If the environment variable is not defined, fall back to a default.
    tableName := os.Getenv(tableNameEnvVar)
    if len(tableName) == 0 {
        tableName = tableNameDefault
    }

    // Get a random fact
    if factId == -1 {
        // Count the number of facts.
        scanResults, err := ddbClient.Scan(context.TODO(), &dynamodb.ScanInput{
            TableName: aws.String(tableName),
            Select: types.SelectCount,
        })
        if err != nil {
            return nil, err
        }
        factCount := int(scanResults.Count)
        if factCount <= 0 {
            return nil, fmt.Errorf("no facts found in dynamodb table %s", tableName) 
        } else if factCount == 1 {
            factId = 0
        } else {
            factId = rand.Intn(factCount-1)
        }
    }

    tableKey, err := attributevalue.Marshal(factId)
    if err != nil {
        return nil, err
    }

    input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"FactId": tableKey,
		},
	}

    result, err := ddbClient.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}

    if result.Item == nil {
        return nil, nil
    }

    fact := new(Fact)
    err = attributevalue.UnmarshalMap(result.Item, fact)
	if err != nil {
		return nil, err
	}

    return fact, nil
}

func processGet(ctx context.Context, factId int) (events.APIGatewayProxyResponse, error) {
	fact, err := getFact(ctx, factId)
	if err != nil {
        log.Printf("Failed to get fact: %s", err)
		return serverError(err)
	}

	if fact == nil {
        log.Printf("nil returned from getFact!")
		return clientError(http.StatusNotFound)
	}

	json, err := json.Marshal(fact)
	if err != nil {
        log.Printf("Failed to json.Marshal(fact): %s", err)
		return serverError(err)
	}
	log.Printf("Successfully fetched fact: %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    switch req.HTTPMethod {
        case "GET":
            factIdStr, ok := req.QueryStringParameters["FactId"]
            factId, err := strconv.Atoi(factIdStr)
            if err != nil || !ok {
                factId = -1
            }
            return processGet(ctx, factId)
        default:
            return clientError(http.StatusMethodNotAllowed)
	}
}

func main() {
	lambda.Start(router)
}