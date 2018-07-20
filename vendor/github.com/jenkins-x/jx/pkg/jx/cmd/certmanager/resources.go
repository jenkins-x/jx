package certmanager

var (
	Cert_manager_certificate = `
apiVersion: certmanager.k8s.io/v1alpha1
kind: Certificate
metadata:
  name: %s
spec:
  secretName: tls-jx-cert
  issuerRef:
    name: %s
    kind: Issuer
  commonName: %s
  acme:
    config:
    - http01:
        ingressClass: nginx
      domains:
      - %s
`

	Cert_manager_issuer_prod = `
apiVersion: certmanager.k8s.io/v1alpha1
kind: Issuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    # The ACME server URL
    server: https://acme-v02.api.letsencrypt.org/directory
    # Email address used for ACME registration
    email: %s
    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt-prod
    # Enable the HTTP-01 challenge provider
    http01: {}
`
	Cert_manager_issuer_stage = `
apiVersion: certmanager.k8s.io/v1alpha1
kind: Issuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    # The ACME server URL
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    # Email address used for ACME registration
    email: %s
    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt-staging
    # Enable the HTTP-01 challenge provider
    http01: {}
`
)
