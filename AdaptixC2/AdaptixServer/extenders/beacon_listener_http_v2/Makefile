all: clean
	@ echo "    * Building beacon_listener_http_v2 plugin"
	@ mkdir dist
	@ cp config.yaml ax_config.axs ./dist/
	@ GOEXPERIMENT=jsonv2,greenteagc go build -buildmode=plugin -ldflags="-s -w" -o ./dist/listener_beacon_http_v2.so pl_main.go pl_encoder.go pl_crypto.go pl_transport.go
	@ echo "      done..."

clean:
	@ rm -rf dist
