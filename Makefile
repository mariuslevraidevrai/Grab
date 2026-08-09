BINARY_NAME=grab
PREFIX?=/usr/local/bin
VERSION?=v0.0.1
BIN_DIR=bin

LDFLAGS=-s -w -X main.Version=$(VERSION)

COLOR_RESET=\033[0m
COLOR_CYAN=\033[36m
COLOR_GREEN=\033[32m
COLOR_RED=\033[31m
COLOR_YELLOW=\033[33m

.PHONY: all build build-all build-linux build-darwin build-windows install uninstall clean

all: build

build:
	@echo -e "$(COLOR_CYAN)-->$(COLOR_RESET) Building $(BINARY_NAME) $(VERSION) (Native)..."
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo -e "$(COLOR_GREEN)[+]$(COLOR_RESET) Build successful: $(BIN_DIR)/$(BINARY_NAME)"

build-amd64:
	@echo -e "$(COLOR_CYAN)-->$(COLOR_RESET) Building Linux x86_64 (amd64)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 .

build-386:
	@echo -e "$(COLOR_CYAN)-->$(COLOR_RESET) Building Linux x86 32-bit (386)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=386 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-386 .

build-arm64:
	@echo -e "$(COLOR_CYAN)-->$(COLOR_RESET) Building Linux ARM64..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 .

build-arm:
	@echo -e "$(COLOR_CYAN)-->$(COLOR_RESET) Building Linux ARM 32-bit (armv7)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm .

build-darwin-amd64:
	@mkdir -p $(BIN_DIR)
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 .

build-darwin-arm64:
	@mkdir -p $(BIN_DIR)
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 .

build-windows-amd64:
	@mkdir -p $(BIN_DIR)
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe .

build-all: build-amd64 build-386 build-arm64 build-arm build-darwin-amd64 build-darwin-arm64 build-windows-amd64
	@echo -e "$(COLOR_GREEN)[+]$(COLOR_RESET) All architecture binaries built in './$(BIN_DIR)'"

install:
	@mkdir -p $(PREFIX)
	@ARCH=$$(uname -m); \
	TARGET_BIN=""; \
	if [ -f "$(BIN_DIR)/$(BINARY_NAME)" ]; then \
		TARGET_BIN="$(BIN_DIR)/$(BINARY_NAME)"; \
	elif [ "$$ARCH" = "x86_64" ] && [ -f "$(BIN_DIR)/$(BINARY_NAME)-linux-amd64" ]; then \
		TARGET_BIN="$(BIN_DIR)/$(BINARY_NAME)-linux-amd64"; \
	elif [ "$$ARCH" = "i386" -o "$$ARCH" = "i686" ] && [ -f "$(BIN_DIR)/$(BINARY_NAME)-linux-386" ]; then \
		TARGET_BIN="$(BIN_DIR)/$(BINARY_NAME)-linux-386"; \
	elif [ "$$ARCH" = "aarch64" -o "$$ARCH" = "arm64" ] && [ -f "$(BIN_DIR)/$(BINARY_NAME)-linux-arm64" ]; then \
		TARGET_BIN="$(BIN_DIR)/$(BINARY_NAME)-linux-arm64"; \
	elif [[ "$$ARCH" == arm* ]] && [ -f "$(BIN_DIR)/$(BINARY_NAME)-linux-arm" ]; then \
		TARGET_BIN="$(BIN_DIR)/$(BINARY_NAME)-linux-arm"; \
	fi; \
	if [ -z "$$TARGET_BIN" ]; then \
		echo -e "$(COLOR_YELLOW)[!]$(COLOR_RESET) No pre-built binary matching architecture ($$ARCH) found in $(BIN_DIR)/. Building native binary..."; \
		go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) . || exit 1; \
		TARGET_BIN="$(BIN_DIR)/$(BINARY_NAME)"; \
	fi; \
	echo -e "$(COLOR_CYAN)-->$(COLOR_RESET) Installing $$TARGET_BIN to $(PREFIX)/$(BINARY_NAME)..."; \
	install -d $(PREFIX); \
	install -m 755 $$TARGET_BIN $(PREFIX)/$(BINARY_NAME); \
	echo -e "$(COLOR_GREEN)[+]$(COLOR_RESET) Installation complete! Executable available as '$(BINARY_NAME)'."

uninstall:
	@echo -e "$(COLOR_RED)-->$(COLOR_RESET) Removing $(BINARY_NAME) from $(PREFIX)..."
	@rm -f $(PREFIX)/$(BINARY_NAME)
	@echo -e "$(COLOR_GREEN)[+]$(COLOR_RESET) Uninstall complete."

clean:
	@echo -e "$(COLOR_RED)-->$(COLOR_RESET) Cleaning up build directory..."
	@rm -rf $(BIN_DIR)
	@echo -e "$(COLOR_GREEN)[+]$(COLOR_RESET) Cleaned."