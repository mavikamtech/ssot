package main

type SqsRecords struct {
	Records []SqsEvent `json:"records"`
}

type SqsEvent struct {
	Attributes SqsAttr `json:"attributes"`
	AwsRegion  *string `json:"awsRegion"`
	Body       *string `json:"body"`
}

type SqsBody struct {
	Type      *string `json:"type"`
	MessageId *string `json:"messageId"`
	TopicArn  *string `json:"topicArn"`
	Subject   *string `json:"subject"`
	Message   *string `json:"message"`
}

type SqsAttr struct {
	ApproximateFirstReceiveTimestamp *string `json:"approximateFirstReceiveTimestamp"`
	ApproximateReceiveCount          *string `json:"approximateReceiveCount"`
	SenderId                         *string `json:"senderId"`
	SentTimestamp                    *string `json:"sentTimestamp"`
}

type SnsRecords struct {
	Records []S3Event `json:"records"`
}

type S3Event struct {
	EventVersion      *string               `json:"eventVersion"`
	EventSource       *string               `json:"eventSource"`
	AwsRegion         *string               `json:"awsRegion"`
	EventTime         *string               `json:"eventTime"`
	EventName         *string               `json:"eventName"`
	UserIdentity      UserIdentityType      `json:"userIdentity"`
	RequestParameters RequestParametersType `json:"responseElements"`
	S3                S3Type                `json:"s3"`
}

type UserIdentityType struct {
	PrincipalId *string `json:"sourceIPAddress"`
}

type RequestParametersType struct {
	SourceIPAddress *string `json:"eventVersion"`
}

type S3Type struct {
	S3SchemaVersion *string  `json:"s3SchemaVersion"`
	ConfigurationId *string  `json:"configurationId"`
	Bucket          S3Bucket `json:"bucket"`
	Object          S3Object `json:"object"`
}

type S3Bucket struct {
	Name          *string           `json:"name"`
	OwnerIdentity OwnerIdentityType `json:"ownerIdentity"`
	Arn           *string           `json:"arn"`
}

type OwnerIdentityType struct {
	PrincipalId *string `json:"principalId"`
}

type S3Object struct {
	Key       *string `json:"key"`
	Size      *int32  `json:"size"`
	Etag      *string `json:"etag"`
	Sequencer *string `json:"sequencer"`
}
