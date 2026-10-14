# The API service, canvas-api. One container for now: the OpenFGA
# sidecar joins it with the storage. Which image it serves is the deployment's
# to say, not this file's.

locals {
  # The memory of the container in MiB, whichever unit it was written in.
  memory_mib = tonumber(regex("^[0-9]+", var.memory)) * (endswith(var.memory, "Gi") ? 1024 : 1)
}

resource "google_cloud_run_v2_service" "api" {
  name     = var.service_name
  location = var.region

  # The SPA calls it from the families' browsers, over the public internet, at
  # its own address on run.app.
  ingress = "INGRESS_TRAFFIC_ALL"

  # Nothing here is lost with the service: the data lives elsewhere, and the
  # project, which must never be recreated, is not described here. The service
  # itself is rebuilt from this file in a minute.
  deletion_protection = false

  # Down to nothing when nobody is asking, and up to a small ceiling, which
  # bounds how fast a burst can spend the free allowance. Set for
  # the whole service: Cloud Run keeps a revision's own scaling only for the
  # services that had it before.
  scaling {
    min_instance_count = 0
    max_instance_count = var.max_instances
  }

  template {
    service_account                  = google_service_account.runtime.email
    timeout                          = var.request_timeout
    max_instance_request_concurrency = var.concurrency

    containers {
      image = var.image

      resources {
        limits = {
          cpu    = var.cpu
          memory = var.memory
        }

        # Billed per request: an instance that is holding no request holds no
        # CPU, so a request finishes its work before it answers.
        cpu_idle = true

        # A starting instance gets the processor at full speed.
        startup_cpu_boost = true
      }

      # An instance gets its first request once its health answers. The probe
      # asks every second from the start, as the sidecar's does: with the
      # platform's ten seconds between tries, an instance that missed the first
      # one would wait ten more, against a cold start of three. Thirty tries
      # leave it half a minute to come up.
      startup_probe {
        http_get {
          path = "/health"
        }

        period_seconds    = 1
        timeout_seconds   = 1
        failure_threshold = 30
      }

      # The runtime is told nine tenths of the memory as the heap its collector
      # keeps under, and the rest is left to what it holds beside the heap:
      # without it, garbage the service has let go of can outgrow the container
      # before the collector looks.
      env {
        name  = "GOMEMLIMIT"
        value = "${floor(local.memory_mib * 9 / 10)}MiB"
      }
    }
  }

  depends_on = [google_project_service.enabled]

  # The deployment owns which image is served, and this configuration owns
  # everything around it: without the first line the two would take turns. The
  # deployment's gcloud also writes itself into the service as its client, which
  # nothing here sets either. Traffic is not described at all: the provider
  # computes it when no block names it, so a rollback by traffic is not undone
  # by the next apply.
  lifecycle {
    ignore_changes = [
      client,
      client_version,
      template[0].containers[0].image,
    ]
  }
}

# Anyone may call it: what decides whether a caller gets anything is the
# service's own checks.
resource "google_cloud_run_v2_service_iam_member" "public" {
  location = google_cloud_run_v2_service.api.location
  name     = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
