# Makefile for building the Manifold C library (manifoldc) from source.
#
# Usage:
#   make build-manifold   # Clone and build manifoldc
#   make install           # Install headers and libraries to PREFIX
#   make clean             # Remove build artifacts
#
# After installing, build Lignin with the manifold tag:
#   CGO_CFLAGS="-I$(PREFIX)/include" CGO_LDFLAGS="-L$(PREFIX)/lib" \
#     go build -tags=manifold ./...

PREFIX       ?= /usr/local
MANIFOLD_DIR ?= _manifold
BUILD_DIR    ?= $(MANIFOLD_DIR)/build
MANIFOLD_REPO = https://github.com/elalish/manifold.git
MANIFOLD_TAG ?= v3.0.1

.PHONY: build-manifold install clean

build-manifold: $(BUILD_DIR)/src/manifoldc/libmanifoldc.so

$(MANIFOLD_DIR):
	git clone --depth 1 --branch $(MANIFOLD_TAG) $(MANIFOLD_REPO) $(MANIFOLD_DIR)

$(BUILD_DIR)/src/manifoldc/libmanifoldc.so: $(MANIFOLD_DIR)
	cmake -S $(MANIFOLD_DIR) -B $(BUILD_DIR) \
		-DCMAKE_BUILD_TYPE=Release \
		-DMANIFOLD_CBIND=ON \
		-DBUILD_SHARED_LIBS=ON \
		-DMANIFOLD_TEST=OFF \
		-DMANIFOLD_PYBIND=OFF
	cmake --build $(BUILD_DIR) --parallel

install: build-manifold
	cmake --install $(BUILD_DIR) --prefix $(PREFIX)
	@echo ""
	@echo "Manifold installed to $(PREFIX)."
	@echo "Build Lignin with:"
	@echo "  go build -tags=manifold ./..."
	@echo ""
	@echo "If installed to a non-standard prefix, set:"
	@echo "  CGO_CFLAGS=\"-I$(PREFIX)/include\""
	@echo "  CGO_LDFLAGS=\"-L$(PREFIX)/lib\""

clean:
	rm -rf $(MANIFOLD_DIR) $(BUILD_DIR)
