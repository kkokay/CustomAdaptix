.PHONY: build clean

BUILD_DIR := dist

build:
	@echo "[*] Building beacon_listener_http_v2..."
	@mkdir -p $(BUILD_DIR)
	@go build -buildvcs=false -buildmode=plugin -ldflags="-s -w" -o $(BUILD_DIR)/listener_beacon_http_v2.so .
	@echo "[+] Build complete: $(BUILD_DIR)/listener_beacon_http_v2.so"

clean:
	@rm -rf $(BUILD_DIR)
	@echo "[+] Cleaned"
