terraform {
  required_providers {
    nixos = {
      source = "corpix/nixos"
      version = "0.0.1"
    }
    vultr = {
      source = "vultr/vultr"
      version = "2.10.1"
    }
  }
}

variable "VULTR_API_KEY" {
  # set with env var TF_VAR_VULTR_API_KEY=...
  type = string
}

##

provider "vultr" {
  api_key = var.VULTR_API_KEY
  rate_limit = 700
  retry_limit = 3
}

provider "nixos" {
  ssh {
    port = 22
    config = {
      # user = "root"
      # identityFile = "/home/user/.ssh/..."
    }
  }
  # address_priority = {
  #   "0.0.0.0/0" = 0,
  #   "::/0"      = 1,
  # }
}

##

resource "vultr_snapshot_from_url" "example" {
  # google drive + https://sites.google.com/site/gdocs2direct/ could be used
  # but at some point in time you could start getting 403 errors
  # use you object storage to serve image.raw or serve with `make serve`

  # url = "http://x.x.x.x:8666/image.raw"

  # this will fail (probably a bug in provider) with:
  ## Error: error creating server: {"error":"Server add failed: Snapshot cannot be used at this time","status":400}
  ## don't know how to fix this, but snapshots are downloaded in background
  ## so you can re-run apply after they will be downloaded
}

resource "vultr_ssh_key" "example" {
  # FIXME: have no success getting openssh keys from vultr cloud-init
  # will hardcode key into ./image.nix
  ## [root@vultr:~]# curl http://169.254.169.254/1.0/meta-data
  ## instance-id
  ## instance-v2-id
  ## mac
  ## local-ipv4
  ## public-ipv4
  ## network_config/content_path
  ## hostname
  ## SUBID
  ## ipv6-addr
  ## ipv6-prefix
  name = "example"
  ssh_key = file("authorized_key")
}

##

resource "vultr_instance" "example" {
  count = 1
  plan = "vc2-1c-1gb" # minimal 5$ instance
  region = "fra"
  label = "example"
  enable_ipv6 = true
  ddos_protection = false
  backups = "disabled"

  snapshot_id = vultr_snapshot_from_url.example.id
  ssh_key_ids = [vultr_ssh_key.example.id]
}

resource "nixos_instance" "example" {
  count = 1
  address = [
    vultr_instance.example[count.index].main_ip,
    vultr_instance.example[count.index].v6_main_ip,
  ]

  configuration = "main.nix"
  settings = jsonencode({
    users = {
      extraUsers = {
	root = {
	  openssh = {
	    authorizedKeys = {
	      keys = [vultr_ssh_key.example.ssh_key]
	    }
	  }
	}
      }
    }
  })
}
