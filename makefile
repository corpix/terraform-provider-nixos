export TF_LOG ?= ERROR

.PHONY: test
test:
	go test -v ./...

.PHONY: test/integration
test/integration:
	nix-shell --run '                                                \
		cd test &&                                               \
		rm -rf .terraform .terraform.lock.hcl terraform.tfstate; \
		terraform init &&                                        \
		terraform apply -auto-approve &&                         \
		jq . terraform.tfstate                                   \
	'
