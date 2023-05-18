.DEFAULT_GOAL := deploy

.PHONY: deploy
deploy: test
	@pulumi up

.PHONY: destroy
destroy:
	@pulumi destroy

.PHONY: test
test: test-integration test-unit

.PHONY: test-integration
test-integration:
	@go test -tags=integration

.PHONY: test-unit
test-unit:
	@go test -tags=unit
