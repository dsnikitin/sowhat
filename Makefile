.PHONY: *

BUILD_VERSION=-ldflags="-X main.buildVersion=v1.0.0"
BUILD_COMMIT=-ldflags="-X main.buildCommit=$(git rev-parse HEAD)"
BUILD_DATE=-ldflags="-X main.buildDate=$(date "+%d-%m-%Y %H:%M:%S")"


build:
	go build $(BUILD_VERSION) $(BUILD_DATE) cmd/sowhat.go