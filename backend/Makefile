.PHONY: run build pack tidy migrate

# CGO_ENABLED=1 is required for the EXI static library (clib/exi_lib.a)
# linked via contract_cgo.go + exi_bridge.cpp.
CGO_ENABLED := 1
export CGO_ENABLED

run:
	go run ./cmd/server/...

build:
	go build -o bin/csms-backend ./cmd/server/...

tidy:
	go mod tidy

migrate:
	mysql -u root -p < db/migrations/001_init.sql

# pack → build + bundle into a deployable tarball.
# Output: bin/csms-backend.tar.gz
# Contents: csms-backend binary, config/, resource/, db/migrations/, run.sh
pack: build
	@rm -rf bin/pack bin/csms-backend.tar.gz
	@mkdir -p bin/pack/bin bin/pack/config bin/pack/resource bin/pack/db
	cp bin/csms-backend bin/pack/bin/
	cp config/config.yaml bin/pack/config/
	@if [ -d resource ]; then cp -r resource/* bin/pack/resource/ 2>/dev/null || true; fi
	cp db/migrations/001_init.sql bin/pack/db/
	@echo '#!/bin/bash' > bin/pack/run.sh
	@echo 'cd "$$(dirname $$0)"' >> bin/pack/run.sh
	@echo 'echo "Starting CSMS Backend..."' >> bin/pack/run.sh
	@echo 'exec ./bin/csms-backend' >> bin/pack/run.sh
	@chmod +x bin/pack/run.sh
	tar -czf bin/csms-backend.tar.gz -C bin/pack .
	@rm -rf bin/pack
	@echo "Package: bin/csms-backend.tar.gz"
