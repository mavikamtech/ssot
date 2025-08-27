# Public Parameters
PROFILE ?= mavik-deployer
REGION  ?= us-east-1
STAGE   ?= dev

BIN_DIR := bin

# =============== hello-world ===============
build-hello-world: $(BIN_DIR)
	@echo "==> Building hello-world..."
	powershell -Command '$$env:GOOS="linux"; $$env:GOARCH="arm64"; $$env:CGO_ENABLED="0"; go build -tags lambda.norpc -o "$(BIN_DIR)/bootstrap" ./hello-world'
	powershell -Command 'Compress-Archive -Path "$(BIN_DIR)/bootstrap" -DestinationPath "$(BIN_DIR)/hello-world.zip" -Force'

deploy-hello-world: build-hello-world
	cd hello-world && npx serverless deploy --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

remove-hello-world:
	cd hello-world && npx serverless remove --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

# =============== poc-etl-api ===============
build-poc-etl-api: $(BIN_DIR)
	@echo "==> Building poc-etl-api..."
	powershell -Command '$$env:GOOS="linux"; $$env:GOARCH="arm64"; $$env:CGO_ENABLED="0"; go build -tags lambda.norpc -o "$(BIN_DIR)/bootstrap" ./poc-etl-api'
	powershell -Command 'Compress-Archive -Path "$(BIN_DIR)/bootstrap" -DestinationPath "$(BIN_DIR)/poc-etl-api.zip" -Force'

deploy-poc-etl-api: build-poc-etl-api
	cd poc-etl-api && npx serverless deploy --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

remove-poc-etl-api:
	cd poc-etl-api && npx serverless remove --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

# =============== poc-etl-sqs ===============
build-poc-etl-sqs: $(BIN_DIR)
	@echo "==> Building poc-etl-sqs..."
	powershell -Command '$$env:GOOS="linux"; $$env:GOARCH="arm64"; $$env:CGO_ENABLED="0"; go build -tags lambda.norpc -o "$(BIN_DIR)/bootstrap" ./poc-etl-sqs'
	powershell -Command 'Compress-Archive -Path "$(BIN_DIR)/bootstrap" -DestinationPath "$(BIN_DIR)/poc-etl-sqs.zip" -Force'

deploy-poc-etl-sqs: build-poc-etl-sqs
	cd poc-etl-sqs && npx serverless deploy --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

remove-poc-etl-sqs:
	cd poc-etl-sqs && npx serverless remove --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)


# General
$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
