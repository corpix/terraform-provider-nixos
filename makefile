export TF_LOG ?= ERROR

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
