# The identities of the service and of its deployment, and the narrow rights
# each holds (spec 13, 15.2).
#
# The pool a deployment's tokens are exchanged in is deliberately not described
# here. It is what this configuration itself is applied through, and a door
# cannot be built by whoever walks in by it: the pool, its provider and the
# identity that runs Terraform are created once, from outside, by
# infra/bootstrap.sh, and named here by id.
#
# Each identity is lent to the jobs of one GitHub environment, by the subject
# GitHub gives such a job, which names the environment. Only main may run in
# either environment, and each environment's jobs reach their own identity
# alone: the delivery's jobs run in prod, the Hosting job in prod-hosting
# (deploy.yml, R58).

locals {
  # Everybody whose token carries one subject, ready for its environment.
  github_subject = "principal://iam.googleapis.com/projects/${data.google_project.this.number}/locations/global/workloadIdentityPools/${var.workload_identity_pool_id}/subject/${var.github_oidc_subject_prefix}"
}

# What the service runs as. It holds no role yet, because the service calls no
# Google API of its own: each grant arrives with the code that first uses it —
# a secret it reads, the traces it sends, the model it calls (RUN.md, rule 13).
resource "google_service_account" "runtime" {
  account_id   = "${var.service_name}-run"
  display_name = "Runtime identity of the ${var.service_name} service"

  depends_on = [google_project_service.enabled]
}

# How a deployment gets in: an identity with just enough rights to push an image
# and roll a revision, borrowed by the delivery's jobs. No service account key
# exists anywhere, so there is none to leak.
resource "google_service_account" "deployer" {
  account_id   = "${var.service_name}-deploy"
  display_name = "Deployment identity of the ${var.service_name} service"

  depends_on = [google_project_service.enabled]
}

resource "google_service_account_iam_member" "deployer_from_github" {
  service_account_id = google_service_account.deployer.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "${local.github_subject}:environment:prod"
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

# The SPA's publisher: Firebase Hosting and nothing else. Its job runs a large
# program of somebody else's, the Firebase CLI, so the job has an environment
# and an identity of its own, and reaches neither the service nor this
# configuration. Hosting takes no custom role and its rights are granted on the
# project: the ready-made Hosting role, and the reading of the project's API
# keys, which the CLI's deploy asks for (Firebase's documentation, read
# 2026-10-14). These two are the only roles the identity that applies this
# configuration may grant on the project (infra/bootstrap.sh, R58).
resource "google_service_account" "hosting" {
  account_id   = "${var.service_name}-hosting"
  display_name = "Publishing identity of the ${var.service_name} SPA"

  depends_on = [google_project_service.enabled]
}

resource "google_service_account_iam_member" "hosting_from_github" {
  service_account_id = google_service_account.hosting.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "${local.github_subject}:environment:prod-hosting"
}

resource "google_project_iam_member" "hosting" {
  for_each = toset([
    "roles/firebasehosting.admin",
    "roles/serviceusage.apiKeysViewer",
  ])

  project = var.project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.hosting.email}"
}
