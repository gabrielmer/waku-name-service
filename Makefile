LIBWAKU_DEP_PATH=$(shell go list -m -f '{{.Dir}}' github.com/waku-org/waku-go-bindings)
LIBWAKU_PATH=$(LIBWAKU_DEP_PATH)/third_party/nwaku/build

CGO_LDFLAGS="-L$(LIBWAKU_PATH) -Wl,-rpath -Wl,$(LIBWAKU_PATH)"
CGO_CFLAGS="-I$(LIBWAKU_PATH)"
BIN=wns-server

all: run

buildlib:
	cd $(LIBWAKU_DEP_PATH) && \
	sudo mkdir -p third_party && \
	sudo chown $(USER) third_party && \
	make -C waku

build:
	CGO_CFLAGS=$(CGO_CFLAGS) CGO_LDFLAGS=$(CGO_LDFLAGS) go build -o $(BIN) main.go

run:
	LD_LIBRARY_PATH=$(LD_LIBRARY_PATH):$(LIBWAKU_PATH) go run main.go

clean:
	rm -f $(BIN)