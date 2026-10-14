# The identities of the service and of its deployment, and the narrow rights
# each holds (spec 13, 15.2).
#
# The pool a deployment's tokens are exchanged in is deliberately not described
# here. It is what this configuration itself is applied through, and a door
# cannot be built by whoever walks in by it: the pool, its provider and the
# identity that runs Terraform are created once, from outside, by
# infra/bootstrap.sh, and named here by id.

# What the service runs as. It holds no role yet, because the service calls no
# Google API of its own: each grant arrives with the code that first uses it —
# a secret it reads, the traces it sends, the model it calls (RUN.md, rule 13).
resource "google_service_account" "runtime" {
  account_id   = "${var.service_name}-run"
  display_name = "Runtime identity of the ${var.service_name} service"

  depends_on = [google_project_service.enabled]
}

# How a deployment gets in: an identity with just enough rights to push an image
# and roll a revision, borrowed through the tokens GitHub issues to one
# repository. No service account key exists anywhere, so there is none to leak.
resource "google_service_account" "deployer" {
  account_id   = "${var.service_name}-deploy"
  display_name = "Deployment identity of the ${var.service_name} service"

  depends_on = [google_project_service.enabled]
}

resource "google_service_account_iam_member" "deployer_from_github" {
  service_account_id = google_service_account.deployer.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${data.google_project.this.number}/locations/global/workloadIdentityPools/${var.workload_identity_pool_id}/attribute.repository/${var.github_repository}"
}

# Push an image into the one repository,
resource "google_artifact_registry_repository_iam_member" "deployer" {
  location   = google_artifact_registry_repository.images.location
  repository = google_artifact_registry_repository.images.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.deployer.email}"
}

# roll a revision of the one service — and only that. The ready-made developer
# role carries sixty permissions, among them deleting the service and opening a
# root shell inside a running container; a deployment needs five.
resource "google_project_iam_custom_role" "deployer" {
  role_id     = "${replace(var.service_name, "-", "")}Deployer"
  title       = "Deploys ${var.service_name}"
  description = "Reads the service and rolls a new revision of it. Nothing else."

  permissions = [
    "run.services.get",
    "run.services.update",
    "run.operations.get",
    "run.revisions.get",
    "run.revisions.list",
  ]

  depends_on = [google_project_service.enabled]
}

resource "google_cloud_run_v2_service_iam_member" "deployer" {
  location = google_cloud_run_v2_service.api.location
  name     = google_cloud_run_v2_service.api.name
  role     = google_project_iam_custom_role.deployer.name
  member   = "serviceAccount:${google_service_account.deployer.email}"
}

# and let that revision run as the runtime identity. Nothing wider: it cannot
# create services, change who may call them, or read a secret.
resource "google_service_account_iam_member" "deployer_runs_as_runtime" {
  service_account_id = google_service_account.runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deployer.email}"
}
