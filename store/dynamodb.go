package store

import (
	"context"
	"os"

	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

/*
 Helpful documentation when working with aws-sdk-go-v2 dynamodb package
 https://dynobase.dev/dynamodb-golang-query-examples
 https://stackoverflow.com/questions/53698997/golang-dynamodb-unmarshallistofmaps-returns-array-of-empty

*/

var (
	DYNAMO_TABLE_NAME = "pocketbook" //dynamo table name
	REGION            = "us-east-2"
)

type DB struct {
	DynamoClient *dynamodb.Client
}

type Record struct {
	Key   string
	Value string
}

type UserRecord struct {
	UserName string   `json:"user_name" dynamodbav:"username"`
	UserData []Record `json:"user_records" dynamodbav:"user_records"`
}

func NewDynamoDB() *DB {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	var dc *dynamodb.Client

	if os.Getenv("ENV") != "PROD" {
		dc = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.EndpointResolver = dynamodb.EndpointResolverFromURL("http://localhost:8000")
		})

		if !devTableExists(dc) {
			prepDevEnv(dc)
		}
	} else {
		dc = dynamodb.NewFromConfig(cfg)
	}

	if dc == nil {
		log.Fatal("Unable to create a dynamodb client")
	}
	return &DB{
		DynamoClient: dc,
	}
}

//------------------------------------- DEV ENV purposes only -------------------------------------
//checks if a dev table exists in the local dynamodb
func devTableExists(dc *dynamodb.Client) bool {
	p := dynamodb.NewListTablesPaginator(dc, nil, func(o *dynamodb.ListTablesPaginatorOptions) {
		o.StopOnDuplicateToken = true
	})

	for p.HasMorePages() {
		out, err := p.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}

		for _, tn := range out.TableNames {
			if tn == DYNAMO_TABLE_NAME {
				return true
			}
		}
	}
	return false
}

//------------------------------------- DEV ENV purposes only -------------------------------------
//if a table doesn't exist in the dev env create one given the schema below
func prepDevEnv(dc *dynamodb.Client) {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("user_name"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("user_name"),
				KeyType:       types.KeyTypeHash,
			},
		},
		TableName: aws.String(DYNAMO_TABLE_NAME),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	}

	_, err := dc.CreateTable(context.TODO(), input)
	if err != nil {
		log.Fatalf("Got error calling CreateTable: %s", err)
	}
}

//Given an array of Data creates a new timeline object for an incident
func (d *DB) CreateUserRecord(username string, userRecords []UserRecord) error {

	list, err := attributevalue.MarshalList(&userRecords)

	if err != nil {
		return err
	}
	_, err = d.DynamoClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(DYNAMO_TABLE_NAME),
		Item: map[string]types.AttributeValue{
			"user_name":    &types.AttributeValueMemberS{Value: username},
			"user_records": &types.AttributeValueMemberL{Value: list},
		},
	})

	if err != nil {
		return err
	}
	return nil //all is good
}

//given an incident id fetches the timline from dynamodb
func (d *DB) GetUserRecords(user_name string) (UserRecord, error) {
	var ur UserRecord

	out, err := d.DynamoClient.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(DYNAMO_TABLE_NAME),
		Key: map[string]types.AttributeValue{
			"user_name": &types.AttributeValueMemberS{Value: user_name},
		},
	})

	if err != nil {
		return ur, err
	}

	//unmarshal the timeline object from dynamodb to a Timeline object before returning to the user
	err = attributevalue.UnmarshalMap(out.Item, &ur)
	if err != nil {
		return ur, err
	}

	return ur, nil
}
