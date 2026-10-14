# Firebase in the project, and the site the SPA is served from.
# Sign-in and App Check join it with the change that brings them. Both resources
# exist in the beta provider alone.

resource "google_firebase_project" "this" {
  provider = google-beta

  depends_on = [google_project_service.enabled]
}

# Named after the project, which is what makes the SPA's address
# https://<project>.web.app. Either way it ends up the project's default
# site: a project given Firebase after 2026-10-15 gets no site until one is
# made, and the first one made becomes the default; one given Firebase before
# that date got its default site at once, under the project's id, and this
# resource takes that site over rather than making it again.
resource "google_firebase_hosting_site" "spa" {
  provider = google-beta

  site_id = var.project_id

  depends_on = [google_firebase_project.this]
}
