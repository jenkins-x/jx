#!/usr/bin/env bash

# vault-auth-setup - A script to setup Vault's Kubernetes auth plugin

##### Functions
create_kube_service_acccount()
{
 read -r -d '' ROLE_BINDING << EOF
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: role-tokenreview-binding
  namespace: $2
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:auth-delegator
subjects:
- kind: ServiceAccount
  name: $1
  namespace: $2
EOF

  kubectl -n "$2" create serviceaccount "$1"
  echo "$ROLE_BINDING" | kubectl apply -n "$2" -f -
}

sa_secret() {
  kubectl -n "$2" get sa "$1" -o jsonpath="{.secrets[*]['name']}"
}

jwt() {
  kubectl -n "$2" get secret "$1" -o jsonpath="{.data.token}" | base64 --decode; echo
}

cert() {
  kubectl -n "$2" get secret "$1" -o jsonpath="{.data['ca\.crt']}" | base64 --decode; echo
}

setup_vault_auth() {
  vault status
  vault auth enable -path="$1" kubernetes
  vault policy write "$2" - <<EOF
path "$4/*" {
   capabilities = ["create", "update", "read", "delete", "list"]
}
EOF
  vault write auth/"$1"/role/"$2" bound_service_account_names="$2"  bound_service_account_namespaces="$3"  policies="$2" ttl=24h
  vault write auth/"$1"/config token_reviewer_jwt="$5" kubernetes_host="$6" kubernetes_ca_cert="$7"

  vault secrets enable -path="$4" kv-v2
}

setup_vault_kv() {
  vault secrets enable -path="$1" kv-v2
}

usage()
{
  echo "usage: vault-auth-setup [-k k8s_url -s service-account -n namespace -a kube-auth-path -e kv-secret-path] | [-h]]"
}

##### Main
if [ -z ${VAULT_ADDR+x} ]; then echo "VAULT_ADDR needs to be set"; exit 1; fi
if [ -z ${VAULT_TOKEN+x} ]; then echo "VAULT_TOKEN needs to be set";  exit 1; fi

k8s_url=
service_account=
namespace=
kube_auth_path="kubernetes"
secret_path="secret"
while [ "$1" != "" ]; do
    case $1 in
        -u | --k8s-url )       shift
                                k8s_url=$1
                                ;;
        -s | --serviceaccount ) shift
                                service_account=$1
                                ;;
        -n | --namespace )      shift
                                namespace=$1
                                ;;
        -a | --auth-path )      shift
                                kube_auth_path=$1
                                ;;
        -e | --secret-path )    shift
                                secret_path=$1
                                ;;
        -h | --help )           usage
                                exit
                                ;;
        * )                     usage
                                exit 1
    esac
    shift
done

create_kube_service_acccount "$service_account" "$namespace"
sa_secret=$(sa_secret "$service_account" "$namespace")
sa_jwt_token=$(jwt "$sa_secret" "$namespace")
sa_cert=$(cert "$sa_secret" "$namespace")

setup_vault_auth "$kube_auth_path" "$service_account" "$namespace" "$secret_path" "$sa_jwt_token" "$k8s_url" "$sa_cert"
setup_vault_kv "$secret_path"

vault write auth/"$kube_auth_path"/login role="$service_account" jwt="$sa_jwt_token"
