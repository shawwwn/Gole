LDFLAGS := -ldflags="-s -w"
SOURCES := main.go common.go cli.go crypt.go kconfig.go holepunch.go server.go client.go
OUT := gole
ifneq (,$(findstring NT,$(shell uname)))
	OUT := $(OUT).exe
endif

default: $(OUT)
$(OUT): $(SOURCES)
	go build $(LDFLAGS) -o $(OUT) $(SOURCES)

clean:
	-rm $(OUT)
	-rm ./client ./server
	-rm ./gtun
	-rm *.exe
