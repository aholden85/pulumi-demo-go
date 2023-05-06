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
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const bucketNameEnvVar = "IMAGES_BUCKET_NAME"
const bucketNameDefault = "xaas-api-assets"
const objectKeyPrefixEnvVar = "IMAGES_OBJECT_PREFIX"
const objectKeyPrefixDefault = "animals/animal/images/"
const objectPublicUrlTemplate = "https://%s.s3.amazonaws.com/%s"

var s3Client s3.Client

type Image struct {
	Url  string    `json:"url"`
	Tags ImageTags `json:"tags"`
}

type ImageTags map[string]string

func init() {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	s3Client = *s3.NewFromConfig(sdkConfig)
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

func getImage(ctx context.Context) (*Image, error) {
	// Grab the name of the image bucket from the environment variables.
	// If the environment variable is not defined, fall back to a default.
	bucketName := os.Getenv(bucketNameEnvVar)
	if len(bucketName) == 0 {
		bucketName = bucketNameDefault
	}

	// Grab the key prefix for the objects in the image bucket from the
	// environment variables. If the environment variable is not defined, fall
	// back to a default.
	objectKeyPrefix := os.Getenv(objectKeyPrefixEnvVar)
	if len(objectKeyPrefix) == 0 {
		objectKeyPrefix = objectKeyPrefixDefault
	}

	// Get a random image
	// Count the number of images.
	objects, err := s3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(objectKeyPrefix),
	})
	if err != nil {
		return nil, err
	}

	imageId := 0
	imageCount := int(objects.KeyCount)
	if imageCount <= 0 {
		return nil, fmt.Errorf(
			"no images found in image bucket %s w/ prefix %s",
			bucketName,
			objectKeyPrefix,
		)
	} else if imageCount > 1 {
		imageId = rand.Intn(imageCount - 1)
	}

	getTagsInput := &s3.GetObjectTaggingInput{
		Bucket: aws.String(bucketName),
		Key:    objects.Contents[imageId].Key,
	}

	rawTags, err := s3Client.GetObjectTagging(context.TODO(), getTagsInput)
	if err != nil {
		return nil, err
	}

	tags := ImageTags{}
	for _, tag := range rawTags.TagSet {
		tags[*tag.Key] = *tag.Value
	}

	return &Image{
		Url: fmt.Sprintf(
			objectPublicUrlTemplate,
			bucketName,
			*objects.Contents[imageId].Key,
		),
		Tags: tags,
	}, nil
}

func processGet(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	image, err := getImage(ctx)
	if err != nil {
		log.Printf("Failed to get image: %s", err)
		return serverError(err)
	}

	if image == nil {
		log.Printf("nil returned from getImage!")
		return clientError(http.StatusNotFound)
	}

	json, err := json.Marshal(image)
	if err != nil {
		log.Printf("Failed to json.Marshal(image): %s", err)
		return serverError(err)
	}
	log.Printf("Successfully fetched image: %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
	}, nil
}

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "GET":
		return processGet(ctx)
	default:
		return clientError(http.StatusMethodNotAllowed)
	}
}

func main() {
	lambda.Start(router)
}
