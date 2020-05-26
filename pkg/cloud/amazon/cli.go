package amazon

import (
	"github.com/jenkins-x/jx/v2/pkg/packages"
)

// InstallAwsIamAuthenticatorWithVersion install a specific version of iam authenticator for AWS
func InstallAwsIamAuthenticatorWithVersion(version string, skipPathScan bool) error {
	return packages.InstallOrUpdateBinary(packages.InstallOrUpdateBinaryOptions{
		Binary:              "aws-iam-authenticator",
		GitHubOrganization:  "",
		DownloadUrlTemplate: "https://amazon-eks.s3-us-west-2.amazonaws.com/{{.version}}/2019-03-27/bin/{{.os}}/{{.arch}}/aws-iam-authenticator",
		Version:             version,
		SkipPathScan:        skipPathScan,
		VersionExtractor:    nil,
	})
}

// InstallAwsIamAuthenticator install iam authenticator for AWS
func InstallAwsIamAuthenticator(skipPathScan bool) error {
	return InstallAwsIamAuthenticatorWithVersion(packages.IamAuthenticatorAwsVersion, skipPathScan)
}

// InstallEksCtlWithVersion install a specific version of eks cli
func InstallEksCtlWithVersion(version string, skipPathScan bool) error {
	return packages.InstallOrUpdateBinary(packages.InstallOrUpdateBinaryOptions{
		Binary:              "eksctl",
		GitHubOrganization:  "weaveworks",
		DownloadUrlTemplate: "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_{{.osTitle}}_{{.arch}}.{{.extension}}",
		Version:             version,
		SkipPathScan:        skipPathScan,
		VersionExtractor:    nil,
		Archived:            true,
	})
}

// InstallEksCtl installs eks cli
func InstallEksCtl(skipPathScan bool) error {
	return InstallEksCtlWithVersion(packages.EksCtlVersion, skipPathScan)
}
