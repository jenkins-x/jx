package vault

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam"
	goformation "github.com/awslabs/goformation/cloudformation"
	"github.com/awslabs/goformation/cloudformation/resources"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const stackNamePrefix = "jenkins-x-vault-stack"

const (
	readCapacityUnits  = 2
	writeCapacityUnits = 2
)

// ResourceCreationOpts The input parameters to create a vault by default on aws
type ResourceCreationOpts struct {
	Region    string
	Domain    string
	Username  string
	TableName string
}

// CreateVaultResources will automatically create the preresiquites for vault on aws
func CreateVaultResources(vaultParams ResourceCreationOpts) (*string, *string, *string, *string, error) {
	log.Logger().Infof("Creating vault presiquite resources with following values, %s, %s, %s, %s",
		util.ColorInfo(vaultParams.Region),
		util.ColorInfo(vaultParams.Domain),
		util.ColorInfo(vaultParams.Username),
		util.ColorInfo(vaultParams.TableName))

	template := goformation.NewTemplate()

	awsDynamoDbKey := "AWSDynamoDBTable"
	dynamoDbTable := createDynamoDbTable(vaultParams.TableName)
	template.Resources[awsDynamoDbKey] = &dynamoDbTable

	awsS3BucketKey := "AWSS3Bucket"
	s3Name, s3Bucket, err := createS3Bucket(vaultParams.Region, vaultParams.Domain)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "Generating the vault cloudformation template failed")
	}
	template.Resources[awsS3BucketKey] = s3Bucket

	awsIamUserKey := "AWSIAMUser"
	iamUser := createIamUser(vaultParams.Username)
	template.Resources[awsIamUserKey] = &iamUser

	awsKmsKey := "AWSKMSKey"
	kmsKey, err := createKmsKey([]string{awsIamUserKey})
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "Generating the vault cloudformation template failed")
	}
	template.Resources[awsKmsKey] = kmsKey

	awsIamPolicy := "AWSIAMPolicy"
	policy := createIamUserPolicy(vaultParams.Username, []string{awsDynamoDbKey, awsS3BucketKey, awsKmsKey, awsIamUserKey})
	template.Resources[awsIamPolicy] = &policy

	log.Logger().Infof("Generating the vault cloudformation template")

	// and also the YAML AWS CloudFormation template
	yaml, err := template.JSON()
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "Generating the vault cloudformation template failed")
	}

	log.Logger().Infof("Generated the vault cloudformation template successfully")

	// had issues with json on resource, temporary workaround
	yamlProcessed := string(yaml)
	yamlProcessed = strings.Replace(yamlProcessed, "\"BUCKET_LIST_JSON\"", `{ "Fn::Sub" : "${`+awsS3BucketKey+`.Arn}" }`, -1)
	yamlProcessed = strings.Replace(yamlProcessed, "\"BUCKET_JSON\"", `{ "Fn::Sub" : "${`+awsS3BucketKey+`.Arn}/*" }`, -1)
	yamlProcessed = strings.Replace(yamlProcessed, "\"KMS_SUB_JSON\"", `{ "Fn::Sub" : "${`+awsKmsKey+`.Arn}" }`, -1)
	yamlProcessed = strings.Replace(yamlProcessed, "\"DYNAMODB_JSON\"", `{ "Fn::Sub" : "arn:aws:dynamodb:${AWS::Region}:${AWS::AccountId}:table/*" }`, -1)
	yamlProcessed = strings.Replace(yamlProcessed, "\"KMS_USER_JSON\"", `[{ "Fn::Sub": "arn:aws:iam::${AWS::AccountId}:root"}, { "Fn::Sub": "arn:aws:iam::${AWS::AccountId}:user/`+vaultParams.Username+`"}]`, -1)

	valueUUID, err := uuid.NewV4()
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "Generating UUID failed")
	}

	// Create dynamic stack name
	stackName := stackNamePrefix + valueUUID.String()[:7]

	runCloudformationTemplate(&yamlProcessed, &stackName)

	kmsID, err := getKmsID(awsKmsKey, stackName)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "Generating the vault cloudformation template failed")
	}

	accessKey, keySecret, err := createAccessKey(vaultParams.Region, vaultParams.Username)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrapf(err, "Generating the vault cloudformation template failed")
	}

	return accessKey, keySecret, kmsID, s3Name, nil
}

func getKmsID(kmsKey string, stackName string) (*string, error) {
	log.Logger().Infof("Retrieving the vault kms id")
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := cloudformation.New(sess)

	desInput := &cloudformation.DescribeStackResourceInput{
		LogicalResourceId: aws.String(kmsKey),
		StackName:         aws.String(stackName),
	}
	output, err := svc.DescribeStackResource(desInput)

	if err != nil {
		return nil, errors.Wrapf(err, "Unable to retrieve kms Id")
	}

	log.Logger().Infof("Retrieved the vault kms id successfully")
	return output.StackResourceDetail.PhysicalResourceId, nil
}

func createDynamoDbTable(tableName string) resources.AWSDynamoDBTable {
	provisionedThroughput := &resources.AWSDynamoDBTable_ProvisionedThroughput{
		ReadCapacityUnits:  readCapacityUnits,
		WriteCapacityUnits: writeCapacityUnits,
	}

	keyList := []resources.AWSDynamoDBTable_KeySchema{
		{
			AttributeName: "Path",
			KeyType:       "HASH",
		},
		{
			AttributeName: "Key",
			KeyType:       "RANGE",
		},
	}

	attributeDefinitions := []resources.AWSDynamoDBTable_AttributeDefinition{
		{
			AttributeName: "Path",
			AttributeType: "S",
		},
		{
			AttributeName: "Key",
			AttributeType: "S",
		},
	}

	configuation := resources.AWSDynamoDBTable{
		TableName:             tableName,
		ProvisionedThroughput: provisionedThroughput,
		KeySchema:             keyList,
		AttributeDefinitions:  attributeDefinitions,
		Tags: []resources.Tag{
			{
				Key:   "Name",
				Value: "vault-dynamo-db-table",
			},
		},
	}

	return configuation
}

func createS3Bucket(region string, domain string) (*string, *resources.AWSS3Bucket, error) {
	valueUUID, err := uuid.NewV4()
	if err != nil {
		return nil, nil, err
	}

	bucketName := "vault-unseal." + region + "." + domain + "." + valueUUID.String()[:7]
	log.Logger().Infof(bucketName)

	bucketConfig := resources.AWSS3Bucket{
		AccessControl: "Private",
		BucketName:    bucketName,
		VersioningConfiguration: &resources.AWSS3Bucket_VersioningConfiguration{
			Status: "Suspended",
		},
	}

	return &bucketName, &bucketConfig, nil
}

func createKmsKey(depends []string) (*resources.AWSKMSKey, error) {
	type Principal struct {
		AWS string
	}

	type PolicyDocument struct {
		Sid       string
		Effect    string
		Action    string
		Resource  string
		Principal Principal
	}

	type PolicyRoot struct {
		Version   string
		Statement []PolicyDocument
	}

	document := PolicyRoot{
		Version: "2012-10-17",
		Statement: []PolicyDocument{
			{
				Sid:    "Enable IAM User Permissions",
				Effect: "Allow",
				Principal: Principal{
					AWS: "KMS_USER_JSON",
				},
				Action:   "kms:*",
				Resource: "*",
			},
		},
	}

	kmsKey := resources.AWSKMSKey{
		Description: "KMS Key for bank vault unseal",
		KeyPolicy:   document,
	}

	kmsKey.SetDependsOn(depends)

	return &kmsKey, nil
}

func createIamUser(username string) resources.AWSIAMUser {
	iamUser := resources.AWSIAMUser{
		UserName: username,
	}
	return iamUser
}

func createIamUserPolicy(username string, depends []string) resources.AWSIAMPolicy {
	type PolicyDocument struct {
		Sid      string
		Effect   string
		Action   []string
		Resource interface{}
	}

	type PolicyRoot struct {
		Version   string
		Statement []PolicyDocument
	}

	policyDocument := resources.AWSIAMPolicy{
		PolicyName: "vault",
		Users:      []string{username},
		PolicyDocument: PolicyRoot{
			Version: "2012-10-17",
			Statement: []PolicyDocument{
				{
					Sid:    "DynamoDB",
					Effect: "Allow",
					Action: []string{
						"dynamodb:DescribeLimits",
						"dynamodb:DescribeTimeToLive",
						"dynamodb:ListTagsOfResource",
						"dynamodb:DescribeReservedCapacityOfferings",
						"dynamodb:DescribeReservedCapacity",
						"dynamodb:ListTables",
						"dynamodb:BatchGetItem",
						"dynamodb:BatchWriteItem",
						"dynamodb:CreateTable",
						"dynamodb:DeleteItem",
						"dynamodb:GetItem",
						"dynamodb:GetRecords",
						"dynamodb:PutItem",
						"dynamodb:Query",
						"dynamodb:UpdateItem",
						"dynamodb:Scan",
						"dynamodb:DescribeTable",
					},
					Resource: "DYNAMODB_JSON",
				},
				{
					Sid:    "S3",
					Effect: "Allow",
					Action: []string{
						"s3:PutObject",
						"s3:GetObject",
					},
					Resource: "BUCKET_JSON",
				},
				{
					Sid:    "S3List",
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
					},
					Resource: "BUCKET_LIST_JSON",
				},
				{
					Sid:    "KMS",
					Effect: "Allow",
					Action: []string{
						"kms:Encrypt",
						"kms:Decrypt",
					},
					Resource: "KMS_SUB_JSON",
				},
			},
		},
	}

	policyDocument.SetDependsOn(depends)

	return policyDocument
}

func runCloudformationTemplate(templateBody *string, stackName *string) error {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create CloudFormation client in region
	svc := cloudformation.New(sess)

	desInput := &cloudformation.DescribeStacksInput{StackName: stackName}

	input := &cloudformation.CreateStackInput{
		TemplateBody: templateBody,
		StackName:    stackName,
		Capabilities: []*string{
			aws.String("CAPABILITY_NAMED_IAM"),
		},
	}

	_, err := svc.CreateStack(input)
	if err != nil {
		return errors.Wrapf(err, "Unable to create vault preresiquite resources")
	}

	// Wait until stack is created
	err = svc.WaitUntilStackCreateComplete(desInput)
	if err != nil {
		return errors.Wrapf(err, "Unable to create vault preresiquite resources")
	}

	log.Logger().Infof("Ran the vault cloudformation template successfully")
	return nil
}

/**
We can create an access key using cloudformation, however this results the secret being logged.
There is supposed to be a way using a custom type but for now using this
*/
func createAccessKey(region string, username string) (*string, *string, error) {
	svc := iam.New(session.New())
	input := &iam.CreateAccessKeyInput{
		UserName: aws.String(username),
	}
	result, err := svc.CreateAccessKey(input)

	if err != nil {
		return nil, nil, errors.Wrapf(err, "Unable to create access key")
	}

	log.Logger().Infof("Created the vault access key successfully")
	return result.AccessKey.AccessKeyId, result.AccessKey.SecretAccessKey, nil
}
