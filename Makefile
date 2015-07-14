CGO_LDFLAGS	:= -l:libploop.a
export CGO_LDFLAGS

all: build

build:
	go build -v

modprobe:
	@for m in ploop pfmt_ploop1 pfmt_raw pio_direct pio_nfs pio_kaio; do \
		lsmod | grep -q $$m && continue; \
		echo "MODPROBE $$m"; \
		modprobe $$m || exit 1;\
	done

test: modprobe
	go test -v .

.PHONY: all build test
