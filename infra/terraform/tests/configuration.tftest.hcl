# The configuration's own rules, checked against mock providers: nothing here
# signs in anywhere or reads a deployment, so the tests run in tf-check and in
# CI with no credential at all. Both providers are mocked; a provider left out
# would be the real one, and a plan with it would call Google.

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789012"
    }
  }
}

mock_provider "google-beta" {}

variables {
  billing_account = "0A1B2C-3D4E5F-6A7B8C"
}

run "no_operator_makes_no_alert" {
  command = plan

  assert {
    condition     = length(google_monitoring_notification_channel.operator) == 0 && length(google_monitoring_alert_policy.load) == 0
    error_message = "With no operator's address, neither the alert nor its channel is made."
  }
}

run "an_operator_gets_the_alert" {
  command = plan

  variables {
    operator_email = "operator@example.com"
  }

  assert {
    condition     = length(google_monitoring_notification_channel.operator) == 1 && length(google_monitoring_alert_policy.load) == 1
    error_message = "With an operator's address, the alert and its channel are made."
  }

  assert {
    condition     = google_monitoring_alert_policy.load[0].conditions[0].condition_threshold[0].threshold_value == 5
    error_message = "300 requests a minute are 5 a second, which is the rate the condition compares."
  }
}

# The size and the start of the service are spec 15.3's, and the runtime is
# told nine tenths of the memory.
run "the_service_is_sized_as_the_spec_says" {
  command = plan

  assert {
    condition     = google_cloud_run_v2_service.api.scaling[0].min_instance_count == 0 && google_cloud_run_v2_service.api.scaling[0].max_instance_count == 1
    error_message = "The service scales from 0 to 1 instance."
  }

  assert {
    condition     = google_cloud_run_v2_service.api.template[0].max_instance_request_concurrency == 40 && google_cloud_run_v2_service.api.template[0].timeout == "30s"
    error_message = "An instance serves 40 requests at a time, each for 30 seconds at most."
  }

  assert {
    condition     = google_cloud_run_v2_service.api.template[0].containers[0].resources[0].startup_cpu_boost && google_cloud_run_v2_service.api.template[0].containers[0].resources[0].cpu_idle
    error_message = "A starting instance gets its CPU boosted, and an idle one holds none."
  }

  assert {
    condition     = google_cloud_run_v2_service.api.template[0].containers[0].startup_probe[0].http_get[0].path == "/health" && google_cloud_run_v2_service.api.template[0].containers[0].startup_probe[0].period_seconds == 1
    error_message = "The startup probe asks /health every second."
  }

  assert {
    condition     = one([for e in google_cloud_run_v2_service.api.template[0].containers[0].env : e.value if e.name == "GOMEMLIMIT"]) == "921MiB"
    error_message = "1Gi of memory tells the runtime 921MiB."
  }
}

run "memory_in_mebibytes_counts_too" {
  command = plan

  variables {
    memory = "512Mi"
  }

  assert {
    condition     = one([for e in google_cloud_run_v2_service.api.template[0].containers[0].env : e.value if e.name == "GOMEMLIMIT"]) == "460MiB"
    error_message = "512Mi of memory tells the runtime 460MiB."
  }
}

run "the_budget_mails_at_half_nine_tenths_and_all" {
  command = plan

  assert {
    condition     = [for rule in google_billing_budget.spend.threshold_rules : rule.threshold_percent] == [0.5, 0.9, 1.0]
    error_message = "The spend alert mails at 50, 90 and 100 percent of the budget (spec 14.4)."
  }
}

run "the_default_sink_keeps_its_filter" {
  command = plan

  assert {
    condition     = google_logging_project_sink.default.name == "_Default" && endswith(google_logging_project_sink.default.destination, "/locations/us-east4/buckets/default-us-east4")
    error_message = "The _Default sink writes to the bucket of the deployment's region."
  }

  # The filter Cloud Logging creates the sink with, as its documentation gives
  # it (read 2026-10-14): everything but the six audit logs the _Required sink
  # takes.
  assert {
    condition     = google_logging_project_sink.default.filter == "NOT LOG_ID(\"cloudaudit.googleapis.com/activity\") AND NOT LOG_ID(\"externalaudit.googleapis.com/activity\") AND NOT LOG_ID(\"cloudaudit.googleapis.com/system_event\") AND NOT LOG_ID(\"externalaudit.googleapis.com/system_event\") AND NOT LOG_ID(\"cloudaudit.googleapis.com/access_transparency\") AND NOT LOG_ID(\"externalaudit.googleapis.com/access_transparency\")"
    error_message = "The _Default sink keeps the filter Cloud Logging created it with, word for word."
  }
}

run "an_image_by_tag_is_refused" {
  command = plan

  variables {
    image = "us-docker.pkg.dev/cloudrun/container/hello:latest"
  }

  expect_failures = [var.image]
}

run "a_billing_account_of_another_shape_is_refused" {
  command = plan

  variables {
    billing_account = "billing-account"
  }

  expect_failures = [var.billing_account]
}

run "an_operator_address_of_another_shape_is_refused" {
  command = plan

  variables {
    operator_email = "operator"
  }

  expect_failures = [var.operator_email]
}

run "a_service_name_too_long_for_its_identities_is_refused" {
  command = plan

  variables {
    service_name = "a-service-name-of-23-ch"
  }

  expect_failures = [var.service_name]
}

run "a_service_name_too_short_for_its_identities_is_refused" {
  command = plan

  variables {
    service_name = "a"
  }

  expect_failures = [var.service_name]
}

run "a_service_name_ending_in_a_hyphen_is_refused" {
  command = plan

  variables {
    service_name = "canvas-"
  }

  expect_failures = [var.service_name]
}

# Each identity is lent to the jobs of one environment, and to nobody else: the
# delivery's to prod, the SPA's publisher to prod-hosting (R58).
run "each_identity_is_lent_to_one_environment" {
  command = plan

  assert {
    condition     = google_service_account_iam_member.deployer_from_github.member == "principal://iam.googleapis.com/projects/123456789012/locations/global/workloadIdentityPools/canvas-api-github/subject/repo:MathTrail@261630499/canvas-api@1390897704:environment:prod"
    error_message = "The deployment identity is lent to the jobs of the environment prod alone."
  }

  assert {
    condition     = google_service_account_iam_member.hosting_from_github.member == "principal://iam.googleapis.com/projects/123456789012/locations/global/workloadIdentityPools/canvas-api-github/subject/repo:MathTrail@261630499/canvas-api@1390897704:environment:prod-hosting"
    error_message = "The publishing identity is lent to the jobs of the environment prod-hosting alone."
  }
}

# The roles the identity that applies this configuration may grant on the
# project, and no others: the condition in infra/bootstrap.sh names the same two.
run "the_publisher_holds_two_project_roles" {
  command = plan

  assert {
    condition     = toset(keys(google_project_iam_member.hosting)) == toset(["roles/firebasehosting.admin", "roles/serviceusage.apiKeysViewer"])
    error_message = "The publishing identity holds Hosting's role and the reading of API keys, and nothing else on the project."
  }
}

run "a_subject_prefix_of_another_repository_is_refused" {
  command = plan

  variables {
    github_oidc_subject_prefix = "repo:MathTrail@261630499/mathtrail-standalone@1373391176"
  }

  expect_failures = [var.github_oidc_subject_prefix]
}

# The identity that applies this configuration may grant on the project only
# the roles its condition in infra/bootstrap.sh lists. A role granted here and
# not listed there would pass every other check and be refused at the apply,
# after the merge (docs/self-hosting.md).
run "the_publisher_holds_what_bootstrap_lets_the_configuration_grant" {
  command = plan

  assert {
    condition     = toset(keys(google_project_iam_member.hosting)) == toset(regexall("roles/[A-Za-z.]+", regex("hasOnly\\(\\[([^\\]]*)\\]\\)", file("../bootstrap.sh"))[0]))
    error_message = "The roles granted on the project are the ones the condition in infra/bootstrap.sh lets the Terraform identity grant: change both together, and run bootstrap again before the merge."
  }
}
