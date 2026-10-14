# Where the project's logs and traces are kept: in its region, as the rest of it
# is. Cloud Logging made the project's own buckets in `global`, and a
# bucket never changes its location, so the logs go to a bucket of this region
# instead, by the sink that filled the default one. The bucket `_Required` — the
# audit of who changed what in the project — stays where it is: neither it nor
# its sink can be changed.

resource "google_logging_project_bucket_config" "logs" {
  project        = var.project_id
  location       = var.region
  bucket_id      = "default-${var.region}"
  retention_days = var.log_retention_days
  description    = "Every log of the project but the audit of its changes, kept in its region."

  depends_on = [google_project_service.enabled]
}

# The `_Default` sink, sent to the bucket above with the filter it was created
# with, word for word: everything the `_Required` sink does not take. The
# configuration takes the sink over rather than creating one, and a destroy
# leaves it where it is.
#
# The bucket is in the sink's own project, and such a sink has no writer
# identity whatever it asks for (Cloud Logging's API reference, read
# 2026-10-14). The provider reads a sink with no writer identity as one with a
# unique identity, so that is what is written here: written false, every plan
# would show a change that no apply makes (the provider's source, 8.3.0).
resource "google_logging_project_sink" "default" {
  name        = "_Default"
  destination = "logging.googleapis.com/projects/${var.project_id}/locations/${var.region}/buckets/${google_logging_project_bucket_config.logs.bucket_id}"
  filter = join(" AND ", [
    "NOT LOG_ID(\"cloudaudit.googleapis.com/activity\")",
    "NOT LOG_ID(\"externalaudit.googleapis.com/activity\")",
    "NOT LOG_ID(\"cloudaudit.googleapis.com/system_event\")",
    "NOT LOG_ID(\"externalaudit.googleapis.com/system_event\")",
    "NOT LOG_ID(\"cloudaudit.googleapis.com/access_transparency\")",
    "NOT LOG_ID(\"externalaudit.googleapis.com/access_transparency\")",
  ])
  unique_writer_identity = true
}

# Traces are kept in the bucket `_Trace`, which the platform makes with the
# first span the service sends, in the place set here. It is set before that
# span, because a bucket keeps its location for good.
resource "google_observability_project_settings" "traces" {
  project                  = var.project_id
  location                 = "global"
  default_storage_location = var.region

  depends_on = [google_project_service.enabled]
}
