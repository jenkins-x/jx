package vault

// secretPath generates a secret path from the secret name for storing in vault
// this just makes sure it gets stored under /secret
func secretPath(secretName string) string {
	return "secret/" + secretName
}

// AdminSecretPath returns the admin secret path for a given admin secret
func AdminSecretPath(secret AdminSecret) string {
	return AdminSecretsPath + string(secret)
}

// GitOpsSecretsPath returns the path of an install secret
func GitOpsSecretPath(secret string) string {
	return GitOpsSecretsPath + secret
}
