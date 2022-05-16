.PHONY: image
image:
	mkdir -p pub
	nixos-generate -f openstack -c ./image.nix | xargs -I {} qemu-img convert {} pub/image.raw

.PHONY: serve
serve:
	cd pub && python -mhttp.server 8666 > serve.log
