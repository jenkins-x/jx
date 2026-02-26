package vault

// AdminSecretPath returns the admin secret path for a given admin secret
func AdminSecretPath(secret AdminSecret) string {
	return AdminSecretsPath + string(secret)
}

// GitOpsSecretsPath returns the path of an install secret
func GitOpsSecretPath(secret string) string {
	return GitOpsSecretsPath + secret
}

// AuthSecretPath returns the path of an auth secret
func AuthSecretPath(secret string) string {
	return AuthSecretsPath + secret
}
