# What this configuration is pinned to, and where its state lives.
#
# The state bucket is not created here and cannot be: a backend has to exist
# before the configuration that would create it is read. infra/bootstrap.sh
# makes it once, and backend.hcl names it at init time.
#
# Two providers at one version. Firebase's resources are in the beta provider
# alone, and the two are released together, so they move together — by hand,
# with the Terraform the development image installs, since Dependabot would
# lock them with a Terraform of its own.

terraform {
  required_version = "1.16.3"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "8.3.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "8.3.0"
    }
  }

  backend "gcs" {}
}

# Every request is counted against this project, whatever project the
# caller's own credentials name. The budget's API and Firebase's refuse a
# caller that signs in as a person otherwise, as a plan run by hand does. It
# takes Service Usage enabled in the project, which infra/bootstrap.sh does
# before anything here runs.
provider "google" {
  project               = var.project_id
  region                = var.region
  billing_project       = var.project_id
  user_project_override = true
}

provider "google-beta" {
  project               = var.project_id
  region                = var.region
  billing_project       = var.project_id
  user_project_override = true
}
