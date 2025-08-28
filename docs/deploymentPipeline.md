# Deployment Pipeline Guide
## Background

We use GitHub Actions + Serverless Framework + Go in the repository to automate the deployment of our Lambda services.

Key features:

✅ Uses OIDC to authenticate with AWS (no long-lived access keys, more secure & compliant)

✅ Supports parallel deployments (up to 20 services at the same time, limited to avoid AWS rate limits)

✅ On push to main, only deploys services that were changed, avoiding unnecessary redeploys

✅ On manual trigger, performs a full deployment of all services

✅ Each service deployment has its own concurrency lock to prevent multiple pipelines from overwriting the same service simultaneously

## Repository Structure
```
sls/golang/
  ├── lambda1/
  │   └── serverless.yml
  ├── lambda2/
  │   └── serverless.yml
  ├── lambda2/
  │   └── serverless.yml
  └── ...
```

Each subdirectory = one Lambda service

Must contain a serverless.yml with:

```
package:
  artifact: ../bin/<service>.zip
functions:
  ...
```
## Workflow Triggers
1. Push (Selective Deploy)

    - Triggered when code is pushed to main and changes include:

        `sls/golang/**`, `go.mod`, `go.sum`

    - The Discover job runs git diff to detect which service directories changed, and only those services are deployed.

    - Unchanged services are skipped.

2. Manual Trigger (Full Deploy)

    - In GitHub Actions → select Deploy Lambdas in Parallel → click Run workflow.

    - On manual runs, discover selects all services and deploys everything.

## Workflow Stages
1. Discover job

    - Scans all subdirectories under sls/golang/ containing a serverless.yml.

    - Decides which services to deploy:

        - push → only changed services

        - workflow_dispatch → all services

    - Outputs a JSON array, e.g.:

        `["poc-etl-api","poc-etl-sqs"]`

2. Deploy job (matrix)

    - Creates one parallel job per service in the discover output.

    - Each job runs:

        - Build Go binary (GOOS=linux GOARCH=arm64)

        - Package into bin/<service>.zip

        - Run npx serverless deploy inside the service directory

    - Limited to max 20 parallel jobs, this cap is set to stay under AWS CloudFormation / Lambda deployment rate limits.

    - With `concurrency: deploy-<service>` so the same service can’t be deployed by multiple pipelines simultaneously

## Rollback

If you need to roll back to a previous version:

1. Go to GitHub Actions → find the target workflow run (the old deployment)

2. Click Re-run jobs

3. The workflow will redeploy the same commit:

    - If the original run was a full deploy, rerunning it will again redeploy all services.

    - If the original run was a partial (selective) deploy, rerunning it will only redeploy the changed services from that commit, effectively rolling back those specific components.

This ensures rollback behavior always matches the scope of the original deployment.

## Local Debugging

To simulate a deployment locally:
```
cd sls/golang/<service>
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o ../bin/bootstrap
(cd ../bin && zip -q <service>.zip bootstrap && rm -f bootstrap)
npx serverless@3 deploy --stage dev --region us-east-1
```
## Notes

- Adding a new service: create a new folder under sls/golang/<service> with a serverless.yml. It will be auto-detected in the next push.

- Changes in go.mod/go.sum: trigger a full redeploy since all services depend on these modules.

- Security: uses AWS OIDC Role for deployment. Do not configure long-lived Access Keys.

- Failure isolation: if one service deployment fails, it won’t block others. Logs for that service are visible in its individual job.
