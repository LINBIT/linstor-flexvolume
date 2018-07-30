PROJECT_NAME = `basename $$PWD`
LATESTTAG=$(shell git describe --abbrev=0 --tags | tr -d 'v')
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

# packaging, you need the packaging branch for these

# we build binary-only packages and use the static binary in this tarball
linstor-flexvolume-$(LATESTTAG).tar.gz: build
	dh_clean || true
	tar --transform="s,^,linstor-flexvolume-$(LATESTTAG)/," --owner=0 --group=0 -czf $@ linstor-flexvolume debian linstor-flexvolume.spec

# consistency with the other linbit projects
debrelease: linstor-flexvolume-$(LATESTTAG).tar.gz
