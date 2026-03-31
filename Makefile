BINARY_NAME=npm-vet
VERSION ?= dev
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
INSTALL_DIR=$(HOME)/.npm-vet

.PHONY: build test clean install uninstall setup cross-compile

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

test:
	go test ./... -v

test-short:
	go test ./... -short

clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*

# Full install + activate: build binary, place it in ~/.npm-vet/, create npm shim, update PATH
install: build
	@mkdir -p $(INSTALL_DIR)/bin
	@cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Installed $(BINARY_NAME) to $(INSTALL_DIR)/$(BINARY_NAME)"
	@$(INSTALL_DIR)/$(BINARY_NAME) setup --apply

# Remove binary + shim
uninstall:
	@$(INSTALL_DIR)/$(BINARY_NAME) teardown 2>/dev/null || true
	@rm -rf $(INSTALL_DIR)
	@echo "✓ Uninstalled npm-vet from $(INSTALL_DIR)"

# Cross-compile for common CI/CD platforms
cross-compile:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .
