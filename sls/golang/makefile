# Public Parameters
PROFILE ?= mavik-deployer
REGION  ?= us-east-1
STAGE   ?= dev

BIN_DIR := bin

# =============== get-token-test ===============
build-get-token-test: $(BIN_DIR)
	@echo "==> Building get-token-test..."
	powershell -Command '$$env:GOOS="linux"; $$env:GOARCH="arm64"; $$env:CGO_ENABLED="0"; go build -tags lambda.norpc -o "$(BIN_DIR)/bootstrap" ./get-token-test'
	powershell -Command 'Compress-Archive -Path "$(BIN_DIR)/bootstrap" -DestinationPath "$(BIN_DIR)/get-token-test.zip" -Force'

deploy-get-token-test: build-get-token-test
	cd get-token-test && npx serverless deploy --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

remove-get-token-test:
	cd get-token-test && npx serverless remove --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

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

	
# =============== poc-etl-powerbi-api ===============
build-poc-etl-powerbi-api: $(BIN_DIR)
	@echo "==> Building poc-etl-powerbi-api..."
	powershell -Command '$$env:GOOS="linux"; $$env:GOARCH="arm64"; $$env:CGO_ENABLED="0"; go build -tags lambda.norpc -o "$(BIN_DIR)/bootstrap" ./poc-etl-powerbi-api'
	powershell -Command 'Compress-Archive -Path "$(BIN_DIR)/bootstrap" -DestinationPath "$(BIN_DIR)/poc-etl-powerbi-api.zip" -Force'

deploy-poc-etl-powerbi-api: build-poc-etl-powerbi-api
	cd poc-etl-powerbi-api && npx serverless deploy --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

remove-poc-etl-powerbi-api:
	cd poc-etl-powerbi-api && npx serverless remove --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

# =============== etl-pbi-loaninfo-sync ===============
build-etl-pbi-loaninfo-sync: $(BIN_DIR)
	@echo "==> Building etl-pbi-loaninfo-sync..."
	powershell -Command '$$env:GOOS="linux"; $$env:GOARCH="arm64"; $$env:CGO_ENABLED="0"; go build -tags lambda.norpc -o "$(BIN_DIR)/bootstrap" ./etl-pbi-loaninfo-sync'
	powershell -Command 'Compress-Archive -Path "$(BIN_DIR)/bootstrap" -DestinationPath "$(BIN_DIR)/etl-pbi-loaninfo-sync.zip" -Force'

deploy-etl-pbi-loaninfo-sync: build-etl-pbi-loaninfo-sync
	cd etl-pbi-loaninfo-sync && npx serverless deploy --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

remove-etl-pbi-loaninfo-sync:
	cd etl-pbi-loaninfo-sync && npx serverless remove --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)


# =============== etl-pbi-loancashflow-sync ===============
build-etl-pbi-loancashflow-sync: $(BIN_DIR)
	@echo "==> Building etl-pbi-loancashflow-sync..."
	powershell -Command '$$env:GOOS="linux"; $$env:GOARCH="arm64"; $$env:CGO_ENABLED="0"; go build -tags lambda.norpc -o "$(BIN_DIR)/bootstrap" ./etl-pbi-loancashflow-sync'
	powershell -Command 'Compress-Archive -Path "$(BIN_DIR)/bootstrap" -DestinationPath "$(BIN_DIR)/etl-pbi-loancashflow-sync.zip" -Force'

deploy-etl-pbi-loancashflow-sync: build-etl-pbi-loancashflow-sync
	cd etl-pbi-loancashflow-sync && npx serverless deploy --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

remove-etl-pbi-loancashflow-sync:
	cd etl-pbi-loancashflow-sync && npx serverless remove --aws-profile $(PROFILE) --region $(REGION) -s $(STAGE)

# General
$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)
