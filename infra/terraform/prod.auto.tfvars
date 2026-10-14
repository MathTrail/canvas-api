# The one deployment this repository delivers to. Terraform loads the file by
# itself, because of its name, and it is public like the rest of the
# repository: nothing here is a secret or names a person (spec 15.2). Changing
# where this repository deploys is a pull request, not a setting in somebody's
# console.
#
# Two values are deliberately not here. The billing account names the account
# that pays, and the operator's address is a person's: both arrive from the
# environment instead, as TF_VAR_billing_account and TF_VAR_operator_email.

# The project is the Firebase project too, and its id names the SPA's address,
# https://mathtrail-canvas-prod.web.app (R55). Every service, the images and the
# logs live in one region (R54).
project_id = "mathtrail-canvas-prod"
region     = "us-east4"

# Whose workflows may deploy: the repository by its name, and its owner by the
# numeric id, which a renamed or deleted account does not hand on to whoever
# takes the name next.
github_repository = "MathTrail/canvas-api"
github_owner_id   = "261630499"

# The pool, created before the first apply, in which those workflows' tokens
# are exchanged for Google ones.
workload_identity_pool_id = "canvas-api-github"

# How GitHub begins the subject of this repository's tokens: its owner and its
# name, each with its numeric id, which no later owner of either name is given.
github_oidc_subject_prefix = "repo:MathTrail@261630499/canvas-api@1390897704"

# What a month of the project may cost before mail goes out, at 50, 90 and 100
# percent of it (spec 14.4). The currency is the billing account's own: a
# budget in another one is refused, and infra/bootstrap.sh compares the two.
budget_amount   = "1"
budget_currency = "USD"
