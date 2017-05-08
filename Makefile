PROJECT_NAME = drbd
MAIN = main.go
VERSION=`git describe --tags --always --dirty`
BUILD_DIR =_build

DIRECTORIES = $(BUILD_DIR)

GO = go
LDFLAGS = -ldflags "-X main.Version=${VERSION}"
BUILD_CMD = build $(LDFLAGS) -o $(BUILD_DIR)/$(PROJECT_NAME) $(PROJECT_NAME)/$(MAIN)

MKDIR = mkdir
MKDIR_FLAGS = -pv

RM = rm
RM_FLAGS = -rvf

.PHONY: make_directories

all: make_directories build

make_directories:
	$(MKDIR) $(MKDIR_FLAGS) $(DIRECTORIES)  

build: make_directories
	$(GO) $(BUILD_CMD)

clean:
	$(RM) $(RM_FLAGS) $(DIRECTORIES)
