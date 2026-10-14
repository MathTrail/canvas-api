#!/usr/bin/env bash
# Creates the few things that cannot be created by the pipeline that needs them:
# the project, the bucket its state lives in, and the federated identity the
# pipeline signs in as. Everything else about the deployment is Terraform, and
# Terraform runs in the pipeline.
#
# Safe to run again: every step checks before it creates, and the pool's
# provider is brought back to the condition written here. What it does not undo
# is a grant: a role taken out of the list below stays granted until it is
# removed by hand (docs/self-hosting.md). It takes back two things alone: the
# binding that once lent the Terraform identity to every job of the
# repository, and the role of the project's policy held under any condition
# but the one written below.
#
# It reads the deployment's facts from infra/terraform/, so there is one place
# where they are written down, and prints at the end the two lines that a
# workflow needs before it can sign in at all. The billing account arrives from
# the environment, and this script never prints it: what it prints may be
# pasted anywhere.

set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tfvars="$here/terraform/prod.auto.tfvars"
backend="$here/terraform/backend.hcl"

# One value out of an HCL file: the first `name = "value"` it finds.
hcl_value() {
    local name="$1" file="$2"
    sed -n "s/^[[:space:]]*${name}[[:space:]]*=[[:space:]]*\"\([^\"]*\)\".*/\1/p" "$file" | head -1
}

require() {
    local name="$1" value="$2"
    if [[ -z "$value" || "$value" == REPLACE* ]]; then
        echo "bootstrap: ${name} is not filled in yet (see ${tfvars##*/} and ${backend##*/})" >&2
        exit 1
    fi
}

# A project just created, an API just enabled and a service account just made
# each take a few seconds to become visible to IAM, and a request made in that
# window is refused as if it were not allowed. So what follows them is tried a
# few times before its error is shown.
retry() {
    local attempt
    for attempt in 1 2 3 4; do
        if "$@" > /dev/null 2>&1; then
            return 0
        fi
        sleep $((attempt * 3))
    done
    "$@" > /dev/null
}

# A read, tried as often as a change above, that hands on what it printed. A
# read that keeps failing stops the script: read as nothing there, it would
# skip a step without a word.
retry_read() {
    local attempt output
    for attempt in 1 2 3 4; do
        if output="$("$@" 2> /dev/null)"; then
            printf '%s\n' "$output"
            return 0
        fi
        sleep $((attempt * 3))
    done
    "$@"
}

# Makes a thing unless it is there, and looks again before every new try: a
# create refused on this side may still have succeeded on the other, and the
# next try then finds it instead of failing on it. The first argument asks
# whether the thing is there, the second makes it.
ensure() {
    local exists="$1" make="$2" attempt
    for attempt in 1 2 3 4; do
        if "$exists" > /dev/null 2>&1 || "$make" > /dev/null 2>&1; then
            return 0
        fi
        sleep $((attempt * 3))
    done
    "$exists" > /dev/null 2>&1 || "$make" > /dev/null
}

if ! command -v gcloud > /dev/null; then
    echo "bootstrap: there is no gcloud here. The development image installs it on x86_64 only; on arm64, run this from Cloud Shell or from Google's google-cloud-cli image (docs/self-hosting.md)." >&2
    exit 1
fi

project="$(hcl_value project_id "$tfvars")"
region="$(hcl_value region "$tfvars")"
repository="$(hcl_value github_repository "$tfvars")"
owner_id="$(hcl_value github_owner_id "$tfvars")"
pool="$(hcl_value workload_identity_pool_id "$tfvars")"
subject_prefix="$(hcl_value github_oidc_subject_prefix "$tfvars")"
currency="$(hcl_value budget_currency "$tfvars")"
bucket="$(hcl_value bucket "$backend")"

require project_id "$project"
require region "$region"
require github_repository "$repository"
require github_owner_id "$owner_id"
require workload_identity_pool_id "$pool"
require github_oidc_subject_prefix "$subject_prefix"
require budget_currency "$currency"
require bucket "$bucket"

# The account that pays is not written down beside the rest of the deployment,
# because that is written down in public. It arrives here the same way it
# arrives in a delivery.
billing="${TF_VAR_billing_account:-}"
if [[ ! "$billing" =~ ^[0-9A-Z]{6}-[0-9A-Z]{6}-[0-9A-Z]{6}$ ]]; then
    echo "bootstrap: TF_VAR_billing_account is not set, or is not the id of a billing account, which reads like 01ABCD-234567-89EFGH." >&2
    exit 1
fi

service_account="canvas-api-tf@${project}.iam.gserviceaccount.com"

# gcloud names the billing account in its errors about it, and a refusal is
# the likeliest thing for it to say here. So a command given the account shows
# what it wrote to stderr with the id replaced by its name: nothing this script
# prints holds it.
billing_hidden() {
    local errors status=0
    errors="$(mktemp)"
    "$@" 2> "$errors" || status=$?
    sed "s/${billing}/<billing account>/g" "$errors" >&2
    rm -f "$errors"
    return "$status"
}

# A budget is kept in the currency the account is billed in, and one in any
# other currency is refused. That refusal would come only at the first apply,
# after a merge; here it comes before anything is created.
echo "==> billing account"
billed_in="$(billing_hidden gcloud billing accounts describe "$billing" --format='value(currencyCode)')"
if [[ -z "$billed_in" ]]; then
    echo "    its currency is not shown: compare it with budget_currency (${currency}) in the billing console"
elif [[ "$billed_in" != "$currency" ]]; then
    echo "bootstrap: the billing account is billed in ${billed_in}, and budget_currency in ${tfvars##*/} is ${currency}." >&2
    exit 1
fi

# What this script makes, each asked about before it is made (ensure, above).
project_exists() { gcloud projects describe "$project"; }
make_project() { gcloud projects create "$project" --name="MathTrail Canvas"; }

bucket_exists() { gcloud storage buckets describe "gs://${bucket}"; }
make_bucket() {
    gcloud storage buckets create "gs://${bucket}" \
        --project="$project" \
        --location="$region" \
        --uniform-bucket-level-access \
        --public-access-prevention
}

pool_exists() {
    gcloud iam workload-identity-pools describe "$pool" --project="$project" --location=global
}
make_pool() {
    gcloud iam workload-identity-pools create "$pool" \
        --project="$project" \
        --location=global \
        --display-name="GitHub Actions"
}

provider_exists() {
    gcloud iam workload-identity-pools providers describe github \
        --project="$project" --location=global --workload-identity-pool="$pool"
}
make_provider() {
    gcloud iam workload-identity-pools providers create-oidc github \
        --project="$project" \
        --location=global \
        --workload-identity-pool="$pool" \
        --display-name="GitHub Actions" \
        --issuer-uri="https://token.actions.githubusercontent.com" \
        --attribute-mapping="$mapping" \
        --attribute-condition="$condition"
}

identity_exists() { gcloud iam service-accounts describe "$service_account" --project="$project"; }
make_identity() {
    gcloud iam service-accounts create canvas-api-tf \
        --project="$project" \
        --display-name="Identity the infrastructure is applied as"
}

echo "==> project ${project}"
ensure project_exists make_project
retry billing_hidden gcloud billing projects link "$project" --billing-account="$billing"

# Without the first six nothing below can be created, and they are the reason
# this script exists rather than one more Terraform file. Observability is
# here for another reason: its settings refuse requests for a while after the
# API is enabled, so it is enabled long before the apply that sets where the
# traces are kept.
echo "==> the APIs this script and the first apply need"
retry gcloud services enable \
    serviceusage.googleapis.com \
    cloudresourcemanager.googleapis.com \
    iam.googleapis.com \
    sts.googleapis.com \
    iamcredentials.googleapis.com \
    storage.googleapis.com \
    observability.googleapis.com \
    --project="$project"

echo "==> state bucket gs://${bucket}"
ensure bucket_exists make_bucket
# Versioning is what lets a state file that was written badly be rolled back.
retry gcloud storage buckets update "gs://${bucket}" --versioning

echo "==> federated pool ${pool}"
ensure pool_exists make_pool

# Who may sign in through the pool at all. In the first part both halves
# matter: the name says which repository, and the numeric owner id says it is
# still ours — an account that is renamed or deleted frees its name for anyone
# to take, and the number goes with it. The second part admits one workflow,
# the delivery, from whichever ref it runs: no other workflow of the
# repository can borrow either identity behind the pool, the release among
# them, whose job builds with third-party packages while it holds an identity
# token of its own for signing. Until the delivery exists, nothing can.
# A provider that exists already is given the same mapping and condition
# again, so that what is written here is what holds.
echo "==> pool provider github"
delivery=".github/workflows/deploy.yml"
mapping="google.subject=assertion.sub,attribute.repository=assertion.repository"
condition="assertion.repository=='${repository}' && assertion.repository_owner_id=='${owner_id}' && assertion.job_workflow_ref.startsWith('${repository}/${delivery}@')"
ensure provider_exists make_provider
retry gcloud iam workload-identity-pools providers update-oidc github \
    --project="$project" \
    --location=global \
    --workload-identity-pool="$pool" \
    --issuer-uri="https://token.actions.githubusercontent.com" \
    --attribute-mapping="$mapping" \
    --attribute-condition="$condition"

echo "==> identity ${service_account}"
ensure identity_exists make_identity

# Enumerated rather than Owner: enough to create everything the configuration
# describes, and nothing beyond this one project. The pool above is deliberately
# not among them — what this identity signs in through is not its to change.
#
# The first seven make the service and what it is deployed with: the APIs and
# the project they belong to, the image registry, Cloud Run, the identities of
# the service and of its deployment, and the deployment's own role of five
# permissions instead of a ready-made one of sixty. Firebase's role adds
# Firebase to the project and makes the SPA's Hosting site. The two Monitoring
# roles make the load alert, where a deployment names its operator. The last
# two keep the logs and the traces in the deployment's region: a log bucket
# and the sink that fills it, and where traces are stored.
#
# None of them may change who holds what in the project itself; the one role
# that may is granted after them, under a condition. Service accounts are the
# exception: the configuration makes them and says who may act as them, and
# the role that allows it allows it for every account of the project, this one
# included. That is the power the bindings at the end guard: only the jobs of
# the delivery's environment, and a pull request's plan, can borrow this
# identity.
echo "==> what it may do"
for role in \
    roles/serviceusage.serviceUsageAdmin \
    roles/browser \
    roles/artifactregistry.admin \
    roles/run.admin \
    roles/iam.serviceAccountAdmin \
    roles/iam.serviceAccountUser \
    roles/iam.roleAdmin \
    roles/firebase.admin \
    roles/monitoring.notificationChannelEditor \
    roles/monitoring.alertPolicyEditor \
    roles/logging.configWriter \
    roles/observability.editor
do
    retry gcloud projects add-iam-policy-binding "$project" \
        --member="serviceAccount:${service_account}" \
        --role="$role" \
        --condition=None \
        --quiet
    echo "    $role"
done

# The state, and the spend alert, sit outside the project's own policy.
retry gcloud storage buckets add-iam-policy-binding "gs://${bucket}" \
    --member="serviceAccount:${service_account}" \
    --role=roles/storage.objectAdmin \
    --quiet
echo "    roles/storage.objectAdmin on gs://${bucket}"

retry billing_hidden gcloud billing accounts add-iam-policy-binding "$billing" \
    --member="serviceAccount:${service_account}" \
    --role=roles/billing.costsManager \
    --quiet
echo "    roles/billing.costsManager on the billing account"

# Who holds what in the project may be changed for two roles alone: the ones
# the SPA's publisher holds, which Hosting grants on the project and on nothing
# narrower. Without the condition this role would make the identity the
# project's owner in all but name; with it, a grant or a revocation of any other
# role is refused (IAM's "Set limits on granting roles", read 2026-10-14). The
# condition goes in from a file, since the commas in it would split it on the
# command line. A role the configuration comes to grant on the project joins
# this list first, in the same change, and the script is run again before the
# merge (docs/self-hosting.md).
#
# IAM tells bindings apart by their condition, so a list written anew would be
# granted beside the one before it, and a role taken out of the list would stay
# grantable under the old binding. So the role is held under this expression
# alone: under any other, or none, it is taken back first, and when it is held
# under this one, nothing is touched. The title and the description grant
# nothing and are not compared, so a change to them alone is not carried to a
# project that holds the role already. A policy read in its first version names
# a role held under a condition with a suffix, so the role is matched as a
# prefix.
admin_role=roles/resourcemanager.projectIamAdmin
expression="api.getAttribute('iam.googleapis.com/modifiedGrantsByRole', []).hasOnly(['roles/firebasehosting.admin', 'roles/serviceusage.apiKeysViewer'])"
condition_file="$(mktemp)"
trap 'rm -f "$condition_file"' EXIT
cat > "$condition_file" <<CONDITION
expression: ${expression}
title: The publisher's roles alone
description: Grants and revokes on the project only the roles the SPA's publisher holds.
CONDITION
held="$(retry_read gcloud projects get-iam-policy "$project" \
    --flatten='bindings[].members' \
    --format='value(bindings.role,bindings.members,bindings.condition.expression)' \
    | awk -F '\t' -v role="$admin_role" -v member="serviceAccount:${service_account}" \
        'index($1, role) == 1 && $2 == member { print "held under: " $3 }')"
if [[ "$held" != "held under: ${expression}" ]]; then
    if [[ -n "$held" ]]; then
        retry gcloud projects remove-iam-policy-binding "$project" \
            --member="serviceAccount:${service_account}" \
            --role="$admin_role" \
            --all \
            --quiet
        echo "    ${admin_role} under another condition: taken back"
    fi
    retry gcloud projects add-iam-policy-binding "$project" \
        --member="serviceAccount:${service_account}" \
        --role="$admin_role" \
        --condition-from-file="$condition_file" \
        --quiet
fi
echo "    ${admin_role}, for the publisher's two roles alone"

# The jobs that apply the configuration run in the environment prod, which only
# main may run in, and a pull request's plan runs in none: GitHub names each in
# the subject of the job's token, and this identity is lent to those two
# subjects alone.
echo "==> who may borrow it"
project_number="$(gcloud projects describe "$project" --format='value(projectNumber)')"
subject="principal://iam.googleapis.com/projects/${project_number}/locations/global/workloadIdentityPools/${pool}/subject/${subject_prefix}"
for member in "${subject}:environment:prod" "${subject}:pull_request"; do
    retry gcloud iam service-accounts add-iam-policy-binding "$service_account" \
        --project="$project" \
        --member="$member" \
        --role=roles/iam.workloadIdentityUser \
        --quiet
    echo "    ${member##*/subject/}"
done

# The binding that came before them lent this identity to every job of the
# repository. It is taken back once the two above hold, and only if it is
# there, so that a run again changes nothing. The policy is read first and
# searched second: inside the test, a read that failed would look like a
# policy without the binding.
everyone="principalSet://iam.googleapis.com/projects/${project_number}/locations/global/workloadIdentityPools/${pool}/attribute.repository/${repository}"
borrowers="$(retry_read gcloud iam service-accounts get-iam-policy "$service_account" --project="$project" \
    --flatten='bindings[].members' --format='value(bindings.members)')"
if grep -qxF "$everyone" <<< "$borrowers"; then
    retry gcloud iam service-accounts remove-iam-policy-binding "$service_account" \
        --project="$project" \
        --member="$everyone" \
        --role=roles/iam.workloadIdentityUser \
        --quiet
    echo "    every job of the repository: taken back"
fi

provider="projects/${project_number}/locations/global/workloadIdentityPools/${pool}/providers/github"

cat <<SUMMARY

Done. Put these two lines in infra/ci.env and commit them:

GCP_WORKLOAD_IDENTITY_PROVIDER=${provider}
GCP_TERRAFORM_SERVICE_ACCOUNT=${service_account}

Left to do once, in the repository's settings rather than here: the secret
TF_VAR_billing_account, the same id this script was given, which every plan and
apply of the delivery reads; and, for the load alert, the secret
TF_VAR_operator_email. Without the address the alert is simply not made.
SUMMARY
