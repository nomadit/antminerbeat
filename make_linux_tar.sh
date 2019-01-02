#!/bin/bash
cd beat;env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
cd ../scanner;env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
cd ..
tar cvzf antminerbeat.tar.gz ./beat/beat ./beat/antminerbeat-real.yml ./beat/antminerbeat-test.yml ./beat/fields.yml ./serial_key ./beat/antminerbeat.service ./scanner/scanner ./scanner/scanner.yaml ./scanner/scanner.service ./install.sh ./uninstall.sh ./run.sh
