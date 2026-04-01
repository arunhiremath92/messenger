# Define the output directory
BIN_DIR := $(CURDIR)/bin

all: clean pgclient mgclient

# Create the bin directory if it doesn't exist
prepare:
	mkdir -p $(BIN_DIR)

pgclient: prepare
	cd $(CURDIR)/postgres-client && go mod tidy
	cd $(CURDIR)/postgres-client && go build -o $(BIN_DIR)/pgclient client/client.go

mgclient: prepare
	cd $(CURDIR)/mongodb-client && go mod tidy
	cd $(CURDIR)/mongodb-client && go build -o $(BIN_DIR)/mgclient client/client.go

clean:
	rm -rf $(BIN_DIR)