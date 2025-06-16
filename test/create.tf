terraform {
  required_providers {
    flashblade = {
      source = "purestorage/flashblade"
    }
  }
}

provider "flashblade" {
  api_token = "T-f2e05918-e7a3-4abd-8e15-52c906463b8d"
  endpoint  = "10.225.112.185"
  insecure  = true
}

resource "flashblade_file_system" "test" {
  name                       = "my-first-terraform-fs"
  provisioned                = 549755813888 # 512 GiB
  hard_limit_enabled         = true
  snapshot_directory_enabled = false
  writable                   = true

  nfs = {
    rules      = "*(rw,no_root_squash)"
    v3_enabled = true
  }

  smb = {
    enabled = true
  }
}

output "filesystem_all" {
  value = flashblade_file_system.test
}
