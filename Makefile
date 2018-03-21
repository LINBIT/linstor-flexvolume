PROJECT_NAME = `basename $$PWD`
VERSION=`git describe --tags --always --dirty`
OS=linux
ARCH=amd64

MAGIC_DIR = /usr/libexec/kubernetes/kubelet-plugins/volume/exec/linbit~${PROJECT_NAME}

GO = go
LDFLAGS = -ldflags "-X main.Version=${VERSION}"

MKDIR = mkdir
MKDIR_FLAGS = -pv

CP = cp
CP_FLAGS = -i

RM = rm
RM_FLAGS = -vf

all: build

get:
	-go get ./... &> /dev/null

build: get
	$(GO) build $(LDFLAGS)

release: get
	GOOS=$(OS) GOARCH=$(ARCH) $(GO) build $(LDFLAGS) -o $(PROJECT_NAME)-$(OS)-$(ARCH)

install:
	$(MKDIR) $(MKDIR_FLAGS) $(MAGIC_DIR)
	$(CP) $(CP_FLAGS) $(PROJECT_NAME) $(MAGIC_DIR)

clean:
	$(RM) $(RM_FLAGS) $(PROJECT_NAME)

distclean: clean
	$(RM) $(RM_FLAGS) $(PROJECT_NAME)-$(OS)-$(ARCH)
