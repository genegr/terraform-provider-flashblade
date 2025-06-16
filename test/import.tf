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

# This empty-looking block tells Terraform where to import the state.
# We will populate it with the resource's current configuration after the import.
resource "flashblade_file_system" "imported_fs" {
  # The name is the most important attribute to match.
  name = "my-first-terraform-fs"
}
