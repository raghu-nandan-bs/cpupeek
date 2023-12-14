SHELL:=/bin/bash

ARCH := $(shell uname -m | sed -e 's/x86_64/x86/' -e 's/aarch64/arm64/')

CC :=  $(or $(CC),gcc)
CLANG := $(or $(CLANG),clang)
# In Archlinux, libelf needs libzstd
LIBELF_LDFLAGS := $(shell pkg-config --static --libs libelf)
CGO_CFLAGS := "-I$(abspath ./build/libbpf)"
CGO_LDFLAGS := "$(abspath ./build/libbpf/libbpf.a)"


cpupeek: cpupeek.bpf.o build/libbpf/libbpf.a
	@CC=$(CC) \
	CGO_CFLAGS="$(CGO_CFLAGS)" \
	CGO_LDFLAGS="$(CGO_LDFLAGS)" \
	CGO_ENABLED=1 \
		go build -tags netgo -ldflags='-s -w -extldflags "-static $(LIBELF_LDFLAGS)"' -o cpupeek

cpupeek.bpf.o: cpupeek.bpf.c build/libbpf/libbpf.a
	$(CLANG) -mcpu=v3 -g -O2 -Wall -Werror -D__TARGET_ARCH_$(ARCH) -I$(PWD)/build/libbpf $(CFLAGS)  -c -target bpf $< -o $@


libbpf: build/libbpf/libbpf.a

build/libbpf/libbpf.a:
	@echo "building $@"
	@if [ ! -d libbpf/src ]; then git submodule update --init; fi # --init --recursive
	@CFLAGS="-fPIC" \
	LD_FLAGS="" \
		make -C libbpf/src \
		BUILD_STATIC_ONLY=1 \
		DESTDIR=$(abspath ./build/libbpf/) \
		OBJDIR=$(abspath ./build/libbpf/obj) \
		INCLUDEDIR= LIBDIR= UAPIDIR= prefix= libdir= \
		install install_uapi_headers

# Generate vmlinux
.PHONY: vmlinux
vmlinux:
	bpftool btf dump file /sys/kernel/btf/vmlinux  format c > vmlinux.h


.PHONY: clean
clean:
	rm -f  *.o
	rm -rf build/

release:
	docker build -t cpupeek .
	mkdir -p release
	DOCKER_ID=$$(docker create cpupeek) && \
		docker cp $${DOCKER_ID}:/src/cpupeek/release .