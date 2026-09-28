all: clean
	@ echo "    * Building listener_beacon_advanced plugin (Advanced ChaCha20-Poly1305 with OPSEC)"
	@ mkdir dist
	@ cp config.yaml ax_config.axs ./dist/
	@ GOEXPERIMENT=jsonv2,greenteagc go build -buildmode=plugin -ldflags="-s -w" -o ./dist/listener_beacon_advanced.so pl_main.go pl_transport.go pl_crypto.go
	@ echo "      done..."

clean:
	@ rm -rf dist
