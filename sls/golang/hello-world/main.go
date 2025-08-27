package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Resp struct {
	Message string `json:"message"`
}

func handler(ctx context.Context, req events.LambdaFunctionURLRequest) (Resp, error) {
	name := req.QueryStringParameters["name"]
	if name == "" {
		name = "world"
	}

	fmt.Println("111TESTER: ", "nihao")
	return Resp{Message: fmt.Sprintf("Hello, %s! From Go Lambda.", name)}, nil
}

func main() {
	lambda.Start(handler)
}
