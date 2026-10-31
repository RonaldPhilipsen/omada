OAPI ?= $(shell go env GOPATH)/bin/oapi-codegen

.PHONY: fetch patch verify configs gen

fetch:
	python3 fetch_spec.py

patch:
	python3 patch_spec.py api-docs.json api-docs.patched.json

verify:
	python3 verify_mapping.py api-docs.patched.json tags.json

configs:
	python3 gen_configs.py

gen: fetch patch verify configs
	for f in configs/*.yaml; do $(OAPI) -config $$f api-docs.patched.json; done
	gofmt -w ../cloud
