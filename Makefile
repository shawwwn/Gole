LDFLAGS := -ldflags="-s -w"
SOURCES := main.go common.go cli.go crypt.go kconfig.go holepunch.go server.go client.go
OUT := gole
ifneq (,$(findstring NT,$(shell uname)))
	OUT := $(OUT).exe
endif
DATE := $(shell date -u +%Y%m%d)

default: $(OUT)
$(OUT): $(SOURCES)
	go build $(LDFLAGS) -o $(OUT) $(SOURCES)

.PHONY: clean
clean:
	-rm $(OUT)
	-rm ./client ./server
	-rm ./gtun
	-rm *.exe
	-rm gole_darwin_amd64
	-rm gole-linux-386 gole-linux-amd64 gole-linux-arm gole-linux-arm64
	-rm gole-linux-mips gole-linux-mipsle
	-rm gole-windows-386.exe gole-windows-amd64.exe
	-rm *.zip

.PHONY: release
release:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o gole-darwin-amd64 $(SOURCES)
	zip gole-darwin-$(DATE).zip gole-darwin-amd64
	rm gole-darwin-amd64

	GOOS=linux GOARCH=386 go build $(LDFLAGS) -o gole-linux-386 $(SOURCES)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o gole-linux-amd64 $(SOURCES)
	GOOS=linux GOARCH=arm go build $(LDFLAGS) -o gole-linux-arm $(SOURCES)
	GOOS=linux GOARCH=mips go build $(LDFLAGS) -o gole-linux-mips $(SOURCES)
	GOOS=linux GOARCH=mipsle go build $(LDFLAGS) -o gole-linux-mipsle $(SOURCES)
	zip gole-linux-$(DATE).zip gole-linux-386 gole-linux-amd64 gole-linux-mips gole-linux-mipsle
	rm gole-linux-386 gole-linux-amd64 gole-linux-mips gole-linux-mipsle

	GOOS=windows GOARCH=386 go build $(LDFLAGS) -o gole-windows-386.exe $(SOURCES)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o gole-windows-amd64.exe $(SOURCES)
	zip gole-windows-$(DATE).zip gole-windows-386.exe gole-windows-amd64.exe
	rm gole-windows-386.exe gole-windows-amd64.exe
