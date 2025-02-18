package conn

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type AWSClient struct {
	SSMClient *ssm.Client
	AccountID string
	Region    string
	Partition string
}

func CreateAWSClient(ctx context.Context) *AWSClient {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(""))
	if err != nil {
		panic(err)
	}
	ssmClient := ssm.NewFromConfig(cfg)

	accountId, partition, _ := getAccountIDAndPartition(ctx, cfg)

	client := &AWSClient{
		SSMClient: ssmClient,
		AccountID: accountId,
		Partition: partition,
		Region:    cfg.Region,
	}

	return client
}

func getAccountIDAndPartition(ctx context.Context, cfg aws.Config) (string, string, error) {
	stsClient := sts.NewFromConfig(cfg)
	result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", "", err
	}
	accountId, partition, err := parseAccountIDAndPartitionFromARN(aws.ToString(result.Arn))
	return accountId, partition, err
}

func parseAccountIDAndPartitionFromARN(inputARN string) (string, string, error) {
	arn, err := arn.Parse(inputARN)
	if err != nil {
		return "", "", fmt.Errorf("parsing ARN (%s): %s", inputARN, err)
	}
	return arn.AccountID, arn.Partition, nil
}
