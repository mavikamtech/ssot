# AWS KMS 
This document describes the setup and usage of a customer-managed AWS KMS key (`mavik-data-kms-key`) to ensure consistent and secure data encryption 

## 1. Overview

This document describes the setup and usage of a customer-managed AWS KMS key (mavik-data-kms-key) to ensure consistent and secure data encryption across Mavik’s SSOT components, including Secrets Manager, DynamoDB, S3, and Lambda.

Using a single KMS key for all sensitive components simplifies security auditing, ensures centralized encryption control, and supports granular access policies through IAM and KMS key policies.

## 2. KMS Key Configuration
### 2.1 Key Details

- **Key alias**: `mavik-data-kms-key`
- **Key ID**: `cb424700-0dd0-4dcb-a77d-13af16579270`
- **Region**: `us-east-1`
- **Key type**: Symmetric (AES-256)
- **Status**: Enabled
- **Rotation**: Enabled, yearly rotation

### 2.2 Purpose

This key is used to encrypt and decrypt all sensitive SSOT data, including:

- Application secrets in AWS Secrets Manager
- Encrypted data in DynamoDB
- Files in S3
- Environment variables and runtime data in Lambda

## 3. Key Policy

The following KMS key policy provides controlled access for both administrators and AWS managed services:

```json
{
  "Version": "2012-10-17",
  "Id": "key-consolepolicy-3",
  "Statement": [
    {
      "Sid": "Enable IAM User Permissions",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::167067248318:root"
      },
      "Action": "kms:*",
      "Resource": "*"
    },
    {
      "Sid": "Allow access for Key Administrators",
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::167067248318:user/mavik-developer",
          "arn:aws:iam::167067248318:user/mavik-deployer"
        ]
      },
      "Action": [
        "kms:Create*",
        "kms:Describe*",
        "kms:Enable*",
        "kms:List*",
        "kms:Put*",
        "kms:Update*",
        "kms:Revoke*",
        "kms:Disable*",
        "kms:Get*",
        "kms:Delete*",
        "kms:TagResource",
        "kms:UntagResource",
        "kms:ScheduleKeyDeletion",
        "kms:CancelKeyDeletion",
        "kms:RotateKeyOnDemand"
      ],
      "Resource": "*"
    },
    {
      "Sid": "Allow use of the key",
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::167067248318:role/ecsTaskExecutionRole",
          "arn:aws:iam::167067248318:role/MavikLambdaRole"
        ]
      },
      "Action": [
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:ReEncrypt*",
        "kms:GenerateDataKey*",
        "kms:DescribeKey"
      ],
      "Resource": "*"
    },
    {
      "Sid": "Allow attachment of persistent resources",
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::167067248318:role/ecsTaskExecutionRole",
          "arn:aws:iam::167067248318:role/MavikLambdaRole"
        ]
      },
      "Action": [
        "kms:CreateGrant",
        "kms:ListGrants",
        "kms:RevokeGrant"
      ],
      "Resource": "*",
      "Condition": {
        "Bool": {
          "kms:GrantIsForAWSResource": "true"
        }
      }
    },
    {
      "Sid": "Allow AWS Managed Services To Use Key",
      "Effect": "Allow",
      "Principal": {
        "Service": [
          "dynamodb.amazonaws.com",
          "s3.amazonaws.com",
          "lambda.amazonaws.com",
          "secretsmanager.amazonaws.com"
        ]
      },
      "Action": [
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:ReEncrypt*",
        "kms:GenerateDataKey*",
        "kms:DescribeKey"
      ],
      "Resource": "*"
    }
  ]
}
```

### Key Principles:

- Root user retains full administrative control
- `mavik-deployer` and `mavik-developer` can manage the key lifecycle
- `MavikLambdaRole` and `ecsTaskExecutionRole` can encrypt/decrypt using the key
- AWS managed services (S3, DynamoDB, Secrets Manager, Lambda) can use the key securely

## 4. Using the Key in AWS Services

### 4.1 Secrets Manager

1. Open Secrets Manager → select your secret → Edit encryption key

2. Choose mavik-data-kms-key from the dropdown.

3. Do not check “Create new version of secret with new encryption key.”

4. Save the changes.

To verify:

```bash
aws secretsmanager describe-secret \
aws secretsmanager describe-secret \
  --secret-id <secret-name> \
  --query "KmsKeyId"
```

**Expected result:**

```
"arn:aws:kms:us-east-1:167067248318:key/cb424700-0dd0-4dcb-a77d-13af16579270"
```

### 4.2 S3

To enable KMS encryption for a bucket:

1. Go to S3 → Bucket → Properties → Default encryption
2. Select “AWS KMS key” and choose mavik-data-kms-key.

All new objects will be automatically encrypted with this key.

### 4.3 DynamoDB
To enable KMS encryption for a table:
1. Go to DynamoDB → Table → Additional settings → Encryption
2. Choose `mavik-data-kms-key` from the dropdown
3. Do not check "Create new version of secret with new encryption key"
4. Save the changes

### 4.4 Lambda

Lambda functions interact with S3 and other AWS services that use KMS encryption.
In the Mavik SSOT setup, S3 is configured with default encryption using the customer-managed key mavik-data-kms-key.
Therefore, Lambda functions do not need to explicitly call KMS in the application code.

## 5. Verification of KMS Usage

You can confirm whether the key is actively used through the following methods:

### 5.1 KMS Console
- Navigate to KMS Console → Monitoring tab
- View metrics for Encrypt, Decrypt, and GenerateDataKey requests

### 5.2 CloudTrail Logs

```bash
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceName,AttributeValue=arn:aws:kms:us-east-1:167067248318:key/cb424700-0dd0-4dcb-a77d-13af16579270 \
  --region us-east-1
```

### 5.3 Service-Level Validation
- Check each resource's "Encryption key" field (Secrets Manager, S3, DynamoDB)
- The key ARN should match your `mavik-data-kms-key`

## 6. Recommended Best Practicesket 

- Enable automatic key rotation for compliance and reduced manual effort
- Audit key usage regularly via CloudTrail or AWS Config
- Use a consistent key alias (`mavik-data-kms-key`) across environments
- Avoid frequent re-encryption of existing resources unless required
- Limit administrator access to `mavik-deployer` and `mavik-developer` roles only

## 7. Summary

This setup ensures all critical SSOT components use a unified encryption strategy under a single customer-managed KMS key (`mavik-data-kms-key`).

**This configuration provides:**

- Centralized control over encryption lifecycle
- Fine-grained IAM permissions
- Seamless interoperability among AWS managed services

With this configuration, Mavik maintains both data security and operational simplicity.