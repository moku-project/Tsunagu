.PHONY: dist dist-clean dist-full dist-slim zip-full zip-slim zip-all

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST_DIR := dist
FULL_DIR := $(DIST_DIR)/tsunagu-$(VERSION)-full
SLIM_DIR := $(DIST_DIR)/tsunagu-$(VERSION)-slim

dist: dist-clean
	@echo "==> building Go backend"
	cd backend && go build -o ../$(DIST_DIR)/tsunagu-server ./cmd/server

	@echo "==> building sandbox shadow jar"
	./gradlew -p sandbox shadowJar

	@echo "==> building jlink runtime"
	./gradlew -p sandbox jlinkRuntime

	@echo "==> assembling dist/ tree"
	mkdir -p $(DIST_DIR)/sandbox/extensions
	cp sandbox/build/libs/sandbox-0.1.0-all.jar $(DIST_DIR)/sandbox/sandbox.jar
	cp -r sandbox/build/runtime $(DIST_DIR)/sandbox/runtime

	@echo "==> dist/ ready:"
	find $(DIST_DIR) -maxdepth 3

dist-full: dist
	@echo "==> assembling full bundle (with JVM)"
	rm -rf $(FULL_DIR)
	mkdir -p $(FULL_DIR)/sandbox/extensions
	cp $(DIST_DIR)/tsunagu-server $(FULL_DIR)/tsunagu-server
	cp $(DIST_DIR)/sandbox/sandbox.jar $(FULL_DIR)/sandbox/sandbox.jar
	cp -r $(DIST_DIR)/sandbox/runtime $(FULL_DIR)/sandbox/runtime
	@echo "==> full bundle ready at $(FULL_DIR)"

dist-slim: dist
	@echo "==> assembling slim bundle (no JVM)"
	rm -rf $(SLIM_DIR)
	mkdir -p $(SLIM_DIR)/sandbox/extensions
	cp $(DIST_DIR)/tsunagu-server $(SLIM_DIR)/tsunagu-server
	cp $(DIST_DIR)/sandbox/sandbox.jar $(SLIM_DIR)/sandbox/sandbox.jar
	@echo "==> slim bundle ready at $(SLIM_DIR)"

zip-full: dist-full
	cd $(DIST_DIR) && zip -r $(notdir $(FULL_DIR)).zip $(notdir $(FULL_DIR))
	@echo "==> $(FULL_DIR).zip ready"

zip-slim: dist-slim
	cd $(DIST_DIR) && zip -r $(notdir $(SLIM_DIR)).zip $(notdir $(SLIM_DIR))
	@echo "==> $(SLIM_DIR).zip ready"

zip-all: zip-full zip-slim
	@echo "==> both bundles ready:"
	ls -la $(DIST_DIR)/*.zip

dist-clean:
	rm -rf $(DIST_DIR)
