.SHELLFLAGS += -e

export TF_LOG ?= ERROR

root       = $(dir $(abspath $(firstword $(MAKEFILE_LIST))))
result     = $(root)/result/libexec/terraform-providers
namespace  = registry.terraform.io/corpix
name       = nixos
version   ?= $(shell git rev-list --tags --max-count=1 | xargs git describe --tags)
target     = linux_amd64
gpg_key   ?= 190E440CECF0D6C28E22C8F7755E11DE93BDB108

provider_root   = $(result)/$(namespace)/$(name)/$(version)/$(target)
provider_binary = $(provider_root)/terraform-provider-$(name)_$(version)

.PHONY: build
build:
	nix build -f ./default.nix              \
		--argstr namespace $(namespace) \
		--argstr name      $(name)      \
		--argstr version   $(version)

.ONESHELL: release
.PHONY: release
release: build
	rm -rf release || true
	mkdir -p release
	cd release
	cp $(provider_binary) terraform-provider-$(name)_v$(version)
	zip terraform-provider-$(name)_$(version)_$(target).zip *
	echo '{ "version": 1, "metadata": { "protocol_versions": ["5.0"] } }' \
		| jq                                                          \
		> terraform-provider-$(name)_$(version)_manifest.json
	shasum -a 256 *.zip > terraform-provider-$(name)_$(version)_SHA256SUMS

.PNESHELL: release/sign
.PHONY: release/sign
release/sign:
	cd release
	gpg --detach-sign --local-user $(gpg_key) terraform-provider-$(name)_$(version)_SHA256SUMS

.PHONY: test
test:
	go test -v ./...

.PHONY: run/sshd
run/sshd:
	sudo docker run --rm -it --net=host nixos/nix:latest                                                        \
		nix-shell -p openssh                                                                                \
		--run '{ grep sshd: /etc/passwd > /dev/null || echo "sshd:x:666:666::/:/bin/bash" >> /etc/passwd; } \
			&& sed -i "s/^root:!/root:/g" /etc/shadow                                                   \
			&& sed -i "s/^root:x/root:/g" /etc/passwd                                                   \
			&& sed -i "s|/bin/bash|"$$(which bash)"|g" /etc/passwd                                      \
			&& mkdir -p /etc/ssh /var/empty                                                             \
			&& cp -Pf /root/.nix-profile/bin/* /usr/bin                                                 \
			&& cd /etc/ssh                                                                              \
			&& ssh-keygen -A                                                                            \
			&& echo -e "PermitEmptyPasswords yes\nPermitRootLogin yes\nUsePAM no\nLogLevel verbose"     \
			>  /etc/ssh/sshd_config                                                                     \
			&& `which sshd` -e -D -p 2222                                                               \
			&  pid=$$!                                                                                  \
			&& stop() { kill -9 $$pid; wait $$pid; }                                                    \
			&& trap stop SIGINT SIGTERM                                                                 \
			&& wait $$pid                                                                               \
		'

.PHONY: test/integration
test/integration:
	nix-shell --run '                                                \
		cd test &&                                               \
		rm -rf .terraform .terraform.lock.hcl terraform.tfstate; \
		terraform init &&                                        \
		terraform apply -auto-approve &&                         \
		jq . terraform.tfstate                                   \
	'
