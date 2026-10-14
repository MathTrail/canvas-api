# What a deployment says about itself. The values of this one are in
# prod.auto.tfvars, which is public; the two that are not public arrive from the
# environment and are marked sensitive, so that a plan, whose output is printed
# where anyone may read it, shows neither (R34).

variable "project_id" {
  description = "The Google Cloud project that holds every resource here, and the Firebase project too. The SPA's address is named after it."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.project_id))
    error_message = "project_id is a project id: 6 to 30 lowercase letters, digits and hyphens, starting with a letter."
  }
}

variable "region" {
  description = "Where the service, its images and its logs live (R54)."
  type        = string
}

variable "billing_account" {
  description = "The billing account the project pays through. The spend alert is created on it. It names the account that pays, so it is not written down beside the deployment's other facts and arrives from the environment, as TF_VAR_billing_account."
  type        = string
  sensitive   = true

  validation {
    condition     = can(regex("^[0-9A-Z]{6}-[0-9A-Z]{6}-[0-9A-Z]{6}$", var.billing_account))
    error_message = "billing_account is the id of a billing account, which reads like 01ABCD-234567-89EFGH, and it comes from TF_VAR_billing_account."
  }
}

variable "github_repository" {
  description = "The repository whose workflows may deploy, as owner/name. Only tokens issued to it are accepted."
  type        = string

  validation {
    condition     = can(regex("^[^/]+/[^/]+$", var.github_repository))
    error_message = "github_repository is written as owner/name."
  }
}

variable "github_owner_id" {
  description = "The numeric id of the organisation that owns the repository. A name can be given up and taken by somebody else; a number cannot. infra/bootstrap.sh makes the federated pool's condition of it, outside this configuration; it is written beside the rest of the deployment's facts so that there is one place to read them from."
  type        = string

  validation {
    condition     = can(regex("^[0-9]+$", var.github_owner_id))
    error_message = "github_owner_id is the numeric id, not the name."
  }
}

variable "workload_identity_pool_id" {
  description = "The federated pool the repository's tokens are exchanged in. It exists before this configuration is ever applied — it is how the apply itself signs in — so it is named here rather than created."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9-]{4,32}$", var.workload_identity_pool_id))
    error_message = "workload_identity_pool_id is 4 to 32 lowercase letters, digits and hyphens."
  }
}

variable "service_name" {
  description = "The name of the Cloud Run service, and the prefix of everything named after it, its identities among them."
  type        = string
  default     = "canvas-api"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{0,20}[a-z0-9]$", var.service_name))
    error_message = "service_name is 2 to 22 lowercase letters, digits and hyphens, starting with a letter and ending with a letter or digit: Cloud Run names a service so, and the identities named after it take up to seven characters more, of 6 to 30."
  }
}

variable "image" {
  description = "The container image a newly created service starts with, by digest. The default is Google's placeholder, which answers every path, its health among them, until the first image of this service is deployed; from then on the deployment owns which image is served, and changing this variable moves nothing."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello@sha256:be0a21e5d7036cc60741cf60ea3b68e169cc2d00f3ad256c94536f32b0a0755c"

  validation {
    condition     = can(regex("@sha256:[0-9a-f]{64}$", var.image))
    error_message = "The image is named by digest, never by tag: a tag can be moved under a running service."
  }
}

# The size of the service (spec 15.3). Each is a literal on one line: the load
# runs read them from here, and the calibration of the MVP changes them here.

variable "max_instances" {
  description = "How many instances may run at once."
  type        = number
  default     = 1

  validation {
    condition     = var.max_instances >= 1
    error_message = "max_instances is at least one."
  }
}

variable "concurrency" {
  description = "How many requests one instance serves at a time."
  type        = number
  default     = 40

  validation {
    condition     = var.concurrency >= 1 && var.concurrency <= 1000
    error_message = "concurrency is between 1 and 1000, the platform's limit."
  }
}

variable "cpu" {
  description = "CPU of the container, in whole vCPUs, written as the platform takes them: 1, 2, or 1000m, 2000m. Allocated only while a request is being served."
  type        = string
  default     = "1"

  validation {
    condition     = can(regex("^([1-9][0-9]*|[1-9][0-9]*000m)$", var.cpu))
    error_message = "cpu is a whole number of vCPUs, at least one: 1, 2, or 1000m, 2000m."
  }
}

variable "memory" {
  description = "Memory of the container, in Mi or Gi. Nine tenths of it is the soft limit the Go runtime is told."
  type        = string
  default     = "1Gi"

  validation {
    condition     = can(regex("^[1-9][0-9]*(Mi|Gi)$", var.memory))
    error_message = "memory is a whole number of Mi or Gi, such as 1Gi: nine tenths of it is worked out from it for the runtime."
  }
}

variable "request_timeout" {
  description = "How long a single request may take before the platform cuts it off."
  type        = string
  default     = "30s"

  validation {
    condition     = can(regex("^[1-9][0-9]*s$", var.request_timeout))
    error_message = "request_timeout is a whole number of seconds, such as 30s."
  }
}

variable "budget_amount" {
  description = "The monthly spend, in whole units of budget_currency, that the spend alert is measured against. An alert and not a cap: nothing stops a project at a number."
  type        = string

  validation {
    condition     = can(regex("^[1-9][0-9]*$", var.budget_amount))
    error_message = "budget_amount is a whole number of units, at least one."
  }
}

variable "budget_currency" {
  description = "The currency of the spend alert, as an ISO 4217 code. It has to be the currency of the billing account, which infra/bootstrap.sh checks."
  type        = string

  validation {
    condition     = can(regex("^[A-Z]{3}$", var.budget_currency))
    error_message = "budget_currency is a three-letter currency code, such as USD."
  }
}

variable "operator_email" {
  description = "The address the load alert is mailed to. Empty means no alert. It is a person's address, so it is not written in this repository: it arrives from the environment, as TF_VAR_operator_email."
  type        = string
  default     = ""
  sensitive   = true

  validation {
    condition     = var.operator_email == "" || can(regex("^[^@[:space:]]+@[^@[:space:]]+\\.[^@[:space:]]+$", var.operator_email))
    error_message = "operator_email is empty, or an email address."
  }
}

variable "load_alert_per_minute" {
  description = "Requests a minute to the service, all its revisions together and on average over five minutes, past which the load alert is mailed."
  type        = number
  default     = 300

  validation {
    condition     = var.load_alert_per_minute > 0
    error_message = "load_alert_per_minute is more than zero."
  }
}

variable "keep_images" {
  description = "How many of the most recent image versions survive the cleanup, whatever their age."
  type        = number
  default     = 5
}

variable "image_max_age" {
  description = "How old an image version may get before the cleanup deletes it, unless it is one of the most recent ones."
  type        = string
  default     = "30d"

  validation {
    condition     = can(regex("^[1-9][0-9]*d$", var.image_max_age))
    error_message = "image_max_age is a whole number of days, such as 30d."
  }
}

variable "log_retention_days" {
  description = "How many days the logs are kept: the platform's default, which the privacy policy names (privacy 4, 8)."
  type        = number
  default     = 30

  validation {
    condition     = var.log_retention_days >= 1 && var.log_retention_days <= 3650
    error_message = "log_retention_days is between 1 and 3650."
  }
}
