#!/usr/bin/env bash
#
# Script for setting up the BDD tests using Terraform to create cluster, booting a Vault based Jenkins X install
#
# Arguments:
# none

set -e

script_dir=$(dirname "$0")
script_dir=$(realpath "$script_dir")

declare -A exported

###############################################################################
# Helper function which exports an evironment variable and keeps track of the
# exported key/value pair.
#
# Arguments:
# $1 name of the envronment variable to export
# $2 value to exort
###############################################################################
function exp() {
  exported["$1"]="$2"
  export "$1"="$2"
}

###############################################################################
# Helper function to print all exported variables using the exp function
###############################################################################
function print_exported() {
  for key in "${!exported[@]}"; do
    value="${!key}"
    if [[ $key == *"TOKEN"* ]] || [[ $key == *"PASSWORD"* ]]; then
      value="***"
    fi
    printf "%-30s  %s\n" "$key" "$value"
  done | sort
}

###############################################################################
# Helper to keep track of called functions
###############################################################################
function exe()
{
  start=$(date +%s)
  echo "###############################################################################"
  echo "\$ $@"
  echo "###############################################################################"
  "$@"
  end=$(date +%s)
  runtime=$((end-start))
  echo "exec time: $(printf '%dh:%dm:%ds\n' $(($runtime/3600)) $(($runtime%3600/60)) $(($runtime%60)))"
  echo -e "\n\n"
}

###############################################################################
# Setting up environment variables
###############################################################################
function setup_env() {
  exp GH_USERNAME "jenkins-x-bot-test"
  exp GH_EMAIL "jenkins-x@googlegroups.com"
  exp GH_OWNER "jenkins-x-bot-test"

  exp REPORTS_DIR "${BASE_WORKSPACE}/build/reports"
  exp GINKGO_ARGS "-v"

  exp JX_HOME "/tmp/jxhome"
  exp KUBECONFIG "${JX_HOME}/config"

  # lets avoid the git/credentials causing confusion during the test
  exp XDG_CONFIG_HOME $JX_HOME

  # setup jx boot parameters
  exp JX_VALUE_ADMINUSER_PASSWORD "$JENKINS_PASSWORD" # pragma: allowlist secret
  exp JX_VALUE_PIPELINEUSER_USERNAME "$GH_USERNAME"
  exp JX_VALUE_PIPELINEUSER_EMAIL "$GH_EMAIL"
  exp JX_VALUE_PIPELINEUSER_TOKEN "$GH_ACCESS_TOKEN"
  exp JX_VALUE_PROW_HMACTOKEN "$GH_ACCESS_TOKEN"

  exp GOOGLE_APPLICATION_CREDENTIALS /secrets/bdd/sa.json
}

###############################################################################
# Setting up git username, email and credentials
###############################################################################
function setup_git() {
  git config --global --add user.name jenkins-x-bot-test
  git config --global --add user.email jenkins-x@googlegroups.com

  mkdir -p $JX_HOME/git
  # replace the credentials file with a single user entry
  echo "https://$GH_USERNAME:$GH_ACCESS_TOKEN@github.com" > $JX_HOME/git/credentials
}

###############################################################################
# Setting up Helm
###############################################################################
function setup_helm() {
  helm init --client-only
  helm repo add jenkins-x https://storage.googleapis.com/chartmuseum.jenkins-x.io
}

###############################################################################
# Authenticate against Google Cloud
###############################################################################
function authenticate() {
  gcloud auth activate-service-account --key-file "$GOOGLE_APPLICATION_CREDENTIALS"
  gcloud auth list
  gcloud config set project jenkins-x-bdd3
  gcloud config get-value project
}

###############################################################################
# Destroy test cluster
###############################################################################
function cluster_destroy()
{
  echo "###############################################################################"
  echo "Cleanup..."
  pushd "$script_dir"/terraform

  # stop at least the Vault instance to prevent Bank-Vault operator to interfere with deletion
  kubectl delete vault "$(yq r jx-requirements.yml 'cluster.clusterName')"

  # explicitly empty buckets (see https://github.com/jenkins-x/jx/issues/7406)
  # see also https://github.com/GoogleCloudPlatform/gsutil/issues/417
  gsutil -m -q rm -r -f "$(yq r jx-requirements.yml 'storage.logs.url')"/** || true
  gsutil -m -q rm -r -f "$(yq r jx-requirements.yml 'storage.reports.url')"/**  || true
  gsutil -m -q rm -r -f "$(yq r jx-requirements.yml 'storage.repository.url')"/**  || true
  gsutil -m -q rm -r -f gs://"$(yq r jx-requirements.yml 'vault.bucket')"/**  || true

  terraform destroy -auto-approve

  popd
}

###############################################################################
# Create test cluster using Terraform and current master of terraform-google-jx
###############################################################################
function create_cluster() {
  git clone https://github.com/jenkins-x/terraform-google-jx.git "$script_dir"/terraform-google-jx

  mkdir "$script_dir"/terraform
  pushd "$script_dir"/terraform

  # the clustername needs to contains branch and build number for the gc jobs to work
  branch=$(echo "${BRANCH_NAME:=unkown}" | tr '[:upper:]' '[:lower:]')
  cluster_name="${branch}-${BUILD_NUMBER:=unkown}-tf-boot"

  cat >main.tf << EOF
module "jx" {
  source  = "../terraform-google-jx"

  gcp_project                 = "jenkins-x-bdd3"
  zone                        = "europe-west1-c"
  cluster_name                = "$cluster_name"
  git_owner_requirement_repos = "jenkins-x-bot-test"
  force_destroy               = true
}

output "jx_requirements" {
  value = module.jx.jx_requirements
}
EOF
  cat main.tf

  terraform init
  terraform apply -auto-approve
  terraform output jx_requirements > jx-requirements.yml
  echo "Logging generated jx-requirements.yml..."
  cat jx-requirements.yml

  # adding cluster labels to ensure proper garbage collection of test cluster
  create_time=$(date '+%a-%b-%d-%Y-%H-%M-%S' | tr '[:upper:]' '[:lower:]')
  gcloud container clusters update "$cluster_name" --zone=europe-west1-c --update-labels "branch=${branch},cluster=tf-boot,create-time=$create_time"
  gcloud container clusters get-credentials --zone=europe-west1-c "$cluster_name"

  popd
}

###############################################################################
# Clone boot config and apply overrides
###############################################################################
function prepare_boot_config() {
  # Use the latest boot config promoted in the version stream instead of master to avoid conflicts during boot, because
  # boot fetches always the latest version available in the version stream.
  git clone  https://github.com/jenkins-x/jenkins-x-versions.git versions
  boot_config_version=$(jx step get dependency-version --host=github.com --owner=jenkins-x --repo=jenkins-x-boot-config --dir versions | sed 's/.*: \(.*\)/\1/')

  # Clone the boot config
  git clone https://github.com/jenkins-x/jenkins-x-boot-config.git boot-source

  pushd boot-source
  # Checkout the the determined version of the boot config
  git checkout tags/v"${boot_config_version}" -b latest-boot-config

  # Copy in the Terraform generated jx-requirements.yml
  cp "$script_dir"/terraform/jx-requirements.yml .
  yq write -i jx-requirements.yml 'vault.disableURLDiscovery' true

  # Copy in the boot template overrides
    cat >env/parameters.yaml << EOF
adminUser:
  username: admin
enableDocker: false
gitProvider: github
gpg: {}
pipelineUser:
  github:
    host: github.com
    username: jenkins-x-bot-test
    email: jenkins-x@googlegroups.com
EOF

  sed -e s/\$VERSION/${VERSION_PREFIX}${VERSION}/g "${script_dir}/../boot-vault.platform.yaml.template" >> env/jenkins-x-platform/values.tmpl.yaml
  echo "env/jenkins-x-platform/values.tmpl.yaml :"
  cat "env/jenkins-x-platform/values.tmpl.yaml"

  sed -e s/\$VERSION/${VERSION_PREFIX}${VERSION}/g "${script_dir}/../boot-vault.prow.yaml.template" >> env/prow/values.tmpl.yaml
  echo "env/prow/values.tmpl.yaml :"
  cat "env/prow/values.tmpl.yaml"

  # We need to use the image from the Pull Request instead of the versions stream, otherwise we are not testing the PR itself
  sed -i "s/builder-go.*/&:$VERSION/g" jenkins-x.yml
  popd
}

###############################################################################
# jx boot
###############################################################################
function jx_boot() {
  pushd boot-source
  jx -b boot
  kubectl get nodes
  popd
}

###############################################################################
# Running the BDD tests using `jx step bdd`.
# The tests will run against the current cluster created by Terraform.
###############################################################################
function run_tests() {
  jx step bdd \
      --versions-repo https://github.com/jenkins-x/jenkins-x-versions.git \
      --gopath /tmp \
      --git-provider github \
      --use-current-team true\
      --git-username "$GH_USERNAME" \
      --git-owner "$GH_OWNER" \
      --git-api-token "$GH_ACCESS_TOKEN" \
      --default-admin-password "$JENKINS_PASSWORD" \
      --no-delete-app \
      --no-delete-repo \
      --tests test-quickstart-golang-http

  if [ $? -eq 0 ]; then
    echo "BDD tests completed successfully. Deleting test cluster via 'terraform destroy'"
    cluster_destroy
  else
    echo "BDD tests failed. Keeping test cluster to be deleted by garbage collection job"
  fi
}

###############################################################################
# Main
###############################################################################
function main() {
  exe echo "Running BDD tests with jx version :$( jx version --short)"

  exe setup_env
  exe print_exported
  exe setup_git
  exe setup_helm

  exe authenticate

  exe create_cluster
  exe prepare_boot_config
  exe jx_boot
  exe run_tests
}

main
