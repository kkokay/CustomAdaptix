all: clean
	@echo "    * Building beacon_listener_advanced (ChaCha20-Poly1305 + OPSEC)"
	@mkdir dist
	@cp config.yaml ax_config.axs go.mod go.sum ./dist/
	@GOEXPERIMENT=jsonv2,greenteagc go build -buildmode=plugin -ldflags="-s -w" -o ./dist/listener_beacon_advanced.so *.go
	@echo "      done..."

clean:
	@rm -rf dist
