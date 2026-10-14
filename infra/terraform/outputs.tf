# What a person, or a delivery, needs to know after an apply.

output "project_id" {
  description = "The project, which is the Firebase project too."
  value       = var.project_id
}

output "service_url" {
  description = "The service's own address on run.app, the one the SPA calls (R55)."
  value       = google_cloud_run_v2_service.api.uri
}

output "hosting_url" {
  description = "The address the SPA is served at (R55)."
  value       = google_firebase_hosting_site.spa.default_url
}

output "image_repository" {
  description = "Where images are pushed. The image of the service is this, a slash, and the name of the binary in it."
  value       = "${google_artifact_registry_repository.images.location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.repository_id}"
}

output "service_name" {
  description = "The name a delivery names when it rolls a new revision."
  value       = google_cloud_run_v2_service.api.name
}

output "region" {
  description = "The region a delivery names alongside it."
  value       = google_cloud_run_v2_service.api.location
}

output "runtime_service_account" {
  description = "The identity the service runs as."
  value       = google_service_account.runtime.email
}

output "deploy_service_account" {
  description = "The identity a delivery borrows to push an image and roll a revision."
  value       = google_service_account.deployer.email
}

output "hosting_service_account" {
  description = "The identity the SPA is published with."
  value       = google_service_account.hosting.email
}

output "log_bucket" {
  description = "The bucket the project's logs are kept in, in its region (R57)."
  value       = google_logging_project_bucket_config.logs.bucket_id
}
