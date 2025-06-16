# Terraform Provider for Pure Storage FlashBlade

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Terraform Version](https://img.shields.io/badge/terraform-~%3E%201.0-blueviolet)

A Terraform provider for managing resources on a Pure Storage FlashBlade.

> **MVP In Progress:** This provider is currently in an early **MVP (Minimum Viable Product)** stage. It is intended for testing and demonstration purposes only and is **not recommended for production use**. The API and resource schemas are subject to change without notice.

## Requirements

*   [Terraform](https://www.terraform.io/downloads.html) >= 1.0
*   [Go](https://golang.org/doc/install) >= 1.18 (to build the provider from source)
*   Access to a Pure Storage FlashBlade with API credentials.

## Getting Started

Since this provider is not yet published to the Terraform Registry, you will need to build it from source and place the binary in a location where Terraform can find it.

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/genegr/terraform-provider-test.git
    cd terraform-provider-test
    ```

2.  **Build the provider:**
    ```sh
    go build -o terraform-provider-flashblade
    ```

3.  **Install the provider:**
    Move the compiled binary into the user plugins directory.
    
    *   **Linux/macOS:**
        ```sh
        mkdir -p ~/.terraform.d/plugins/registry.terraform.io/genegr/flashblade/1.0.0/$(go env GOOS)_$(go env GOARCH)
        mv terraform-provider-flashblade ~/.terraform.d/plugins/registry.terraform.io/genegr/flashblade/1.0.0/$(go env GOOS)_$(go env GOARCH)/
        ```

## Usage Example

Here is a basic example of how to configure the provider and create a filesystem.

Create a `main.tf` file with the following content:

```hcl
# main.tf

terraform {
  required_providers {
    flashblade = {
      source  = "registry.terraform.io/genegr/flashblade"
      version = "1.0.0"
    }
  }
}

provider "flashblade" {
  # Your FlashBlade management IP or FQDN
  endpoint   = "192.168.1.10"
  
  # Best practice is to use an environment variable for the API token
  # export PURE_API_TOKEN="your-api-token"
  api_token  = "T-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

resource "flashblade_filesystem" "my_first_fs" {
  name = "my-test-filesystem"

  nfs = {
    v3_enabled = true
    v41_enabled = true
  }

  provisioned = 21474836480 # 20 GiB in bytes
}
