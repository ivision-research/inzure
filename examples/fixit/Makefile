# This Makefile is just intended to ensure you're building both the plugins
# and the binary at the same time to ensure they're both built with the same
# version of the inzure library.
#
# This may not be an issue if you're not directly working on the inzure library
# code, but it is a convenience.
GO=go
BUILD=$(GO) build
INSTALL=$(GO) install

CMD_DIR=.
CMD_GO_FILES=$(shell find $(CMD_DIR) -name '*.go')

PLUGIN_DIR=plugins/fix_redis
PLUGIN_FNAME=fix_redis.so
PLUGIN_OUT=$(PLUGIN_DIR)/$(PLUGIN_FNAME)
PLUGIN_OPTS=-buildmode=plugin -o $(PLUGIN_OUT)
PLUGIN_GO_FILES=$(shell find $(PLUGIN_DIR) -name '*.go')

.PHONY: all cmd plugin

all: cmd plugin

plugin: $(PLUGIN_GO_FILES)
	$(BUILD) $(PLUGIN_OPTS) $^

cmd: $(CMD_GO_FILES)
	$(BUILD)
