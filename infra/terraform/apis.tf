# The APIs this project is allowed to use, and the project itself as the rest
# of the configuration reads it. Service Usage and Resource Manager are not in
# the list, because they are what enables the rest: infra/bootstrap.sh turns
# them on before this configuration can be applied at all. The APIs of what
# comes later — Secret Manager, the models, App Check — are turned on by the
# change that first uses them (RUN.md, rule 13).

locals {
  apis = [
    # The service.
    "run.googleapis.com",
    # Where its images are kept.
    "artifactregistry.googleapis.com",
    # The identity a deployment arrives as: a federated pool, the token exchange
    # that turns a GitHub token into a Google one, and the impersonation that
    # follows.
    "iam.googleapis.com",
    "sts.googleapis.com",
    "iamcredentials.googleapis.com",
    # Firebase in the project, and the Hosting site the SPA is served from (R10).
    "firebase.googleapis.com",
    "firebasehosting.googleapis.com",
    # The spend alert, and the load alert.
    "billingbudgets.googleapis.com",
    "monitoring.googleapis.com",
    # Where the logs and the traces are kept (R57). The bootstrap turns
    # Observability on already, long before its settings are written; it is
    # named here as well, because this is the list of what the configuration
    # stands on.
    "logging.googleapis.com",
    "observability.googleapis.com",
  ]
}

resource "google_project_service" "enabled" {
  for_each = toset(local.apis)

  service = each.value

  # Switching an API off is not the opposite of switching it on: it would break
  # whatever else in the project still uses it, and the resources that needed it
  # here are gone by the time this would run.
  disable_on_destroy = false
}

data "google_project" "this" {
  project_id = var.project_id
}
