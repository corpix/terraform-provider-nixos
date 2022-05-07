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

provider "nixos" {
  ssh {
    config = {
      user = "root"
    }
  }
  address_priority = {
    "0.0.0.0/0" = 1,
    "::/0"      = 0,
  }
}

provider "vultr" {
  api_key = var.VULTR_API_KEY
  rate_limit = 700
  retry_limit = 3
}

##

resource "vultr_iso_private" "example" {
  url = "https://channels.nixos.org/nixos-21.11/latest-nixos-minimal-x86_64-linux.iso"
}

resource "vultr_ssh_key" "example" {
  name = "example"
  ssh_key = "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBACa4D4ycVdMtyIt1WUeoG3S/cdCARlyffhn6LsogFLHURvKtoMVV4cgZBrexju4SjpO/nAlHio8y8T1U0nV5WKDJAAIH0PhPt79HWQOi6HB4d/7UUncMndktyVYar0Mneir/Ci2yQEVmq6vYKKPTuwVynCB2r6yG1IzD1rhFEAG5OUeSg== example@localhost"
}

resource "vultr_instance" "example" {
  count = 1
  plan = "vc2-1c-1gb" # minimal 5$ instance
  region = "fra"
  iso_id = vultr_iso_private.example.id
  ssh_key_ids = [vultr_ssh_key.example.id]
  label = "example"
  enable_ipv6 = true
  ddos_protection = false
  backups = "disabled"
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
	      keys = [
		vultr_ssh_key.example.ssh_key
	      ]
	    }
	  }
	}
      }
    }
  })
}
