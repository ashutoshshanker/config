RM=rm -f
DESTDIR=$(SR_CODE_BASE)/snaproute/src/out/bin
PARAMSDIR=$(DESTDIR)/params
SYSPROFILE=$(DESTDIR)/sysprofile
MKDIR=mkdir -p
RSYNC=rsync -rupE
GOLDFLAGS=-r /opt/flexswitch/sharedlib
SRCS=main.go
COMP_NAME=confd

all: gencode exe install 

exe: $(SRCS)
	go build -ldflags="$(GOLDFLAGS)" -o $(DESTDIR)/$(COMP_NAME) $(SRCS)
	$(SR_CODE_BASE)/snaproute/src/config/docgen/gendoc.sh

install:
	 @$(MKDIR) $(PARAMSDIR)
	 @$(MKDIR) $(SYSPROFILE)
	 @$(RSYNC) docsui $(PARAMSDIR)
	 @echo $(DESTDIR)
	 install params/* $(PARAMSDIR)/
	 install $(SR_CODE_BASE)/snaproute/src/models/objectconfig.json $(PARAMSDIR)
	 install $(SR_CODE_BASE)/snaproute/src/models/genObjectConfig.json $(PARAMSDIR)
	 install $(SR_CODE_BASE)/snaproute/src/models/systemProfile.json $(SYSPROFILE)

fmt: $(SRCS)
	 go fmt $(SRCS)

gencode:
	$(SR_CODE_BASE)/reltools/codegentools/gencode.sh

guard:
ifndef SR_CODE_BASE
	 $(error SR_CODE_BASE is not set)
endif

clean:guard
	 $(RM) $(DESTDIR)/$(COMP_NAME) 
