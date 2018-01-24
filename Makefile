PROJECT_NAME = `basename $$PWD`
VERSION=`git describe --tags --always --dirty`
MAGIC_DIR = /usr/libexec/kubernetes/kubelet-plugins/volume/exec/linbit~${PROJECT_NAME}

GO = go
LDFLAGS = -ldflags "-X main.Version=${VERSION}"
BUILD_CMD = build $(LDFLAGS) 

MKDIR = mkdir
MKDIR_FLAGS = -pv

CP = cp
CP_FLAGS = -i

RM = rm
RM_FLAGS = -vf

all: build

build:
	$(GO) $(BUILD_CMD)

install:
	$(MKDIR) $(MKDIR_FLAGS) $(MAGIC_DIR)
	$(CP) $(CP_FLAGS) $(PROJECT_NAME) $(MAGIC_DIR)

clean:
	$(RM) $(RM_FLAGS) $(PROJECT_NAME)
