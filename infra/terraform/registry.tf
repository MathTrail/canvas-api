# Where the images live. Storage is the only thing here that grows on its own,
# so the repository is told what to forget.

resource "google_artifact_registry_repository" "images" {
  location      = var.region
  repository_id = var.service_name
  format        = "DOCKER"
  description   = "Container images of the ${var.service_name} service"

  # The most recent few are what a rollback reaches for.
  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"

    most_recent_versions {
      keep_count = var.keep_images
    }
  }

  # Everything older goes, whether it was ever tagged or not. Keeping wins over
  # deleting, so the policy above is what saves a recent image from this one.
  cleanup_policies {
    id     = "delete-old"
    action = "DELETE"

    condition {
      older_than = var.image_max_age
    }
  }

  # A policy that only reports what it would have deleted deletes nothing.
  cleanup_policy_dry_run = false

  depends_on = [google_project_service.enabled]
}
