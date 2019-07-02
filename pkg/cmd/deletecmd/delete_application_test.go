package deletecmd_test

import (
	"fmt"
	"github.com/jenkins-x/golang-jenkins"
	"github.com/jenkins-x/jx/pkg/cmd/clients/mocks"
	"github.com/jenkins-x/jx/pkg/cmd/deletecmd"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/testhelpers"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/resources/mocks"
	"github.com/petergtz/pegomock"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestDeleteApplicationInJenkins(t *testing.T) {
	pegomock.RegisterMockTestingT(t)
	t.Parallel()

	testOrgNameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	testRepoNameUUID, err := uuid.NewV4()
	assert.NoError(t, err)
	testOrgName := testOrgNameUUID.String()
	testRepoName := testRepoNameUUID.String()

	mockFactory := clients_test.NewMockFactory()
	commonOpts := opts.NewCommonOptionsWithFactory(mockFactory)

	testhelpers.ConfigureTestOptionsWithResources(&commonOpts,
		[]runtime.Object{},
		[]runtime.Object{
			kube.NewPermanentEnvironment("EnvWhereApplicationIsDeployed"),
		},
		gits.NewGitLocal(),
		nil,
		helm_test.NewMockHelmer(),
		resources_test.NewMockInstaller(),
	)
	testhelpers.MockFactoryWithKubeClients(mockFactory, &commonOpts)
	kubeClient, _, _ := mockFactory.CreateKubeClient()

	jenkinsClient := clients_test.NewMockJenkinsClient()
	pegomock.When(mockFactory.CreateJenkinsClient(kubeClient, "jx", commonOpts.In, commonOpts.Out, commonOpts.Err)).ThenReturn(pegomock.ReturnValue(jenkinsClient), pegomock.ReturnValue(nil))
	job := gojenkins.Job{
		Name:     testRepoName,
		FullName: fmt.Sprintf("%s/%s", testOrgName, testRepoName),
		Class:    "org.jenkinsci.plugins.workflow.multibranch.WorkflowMultiBranchProject",
	}
	jobs := []gojenkins.Job{
		job,
	}
	pegomock.When(jenkinsClient.GetJobs()).ThenReturn(pegomock.ReturnValue(jobs), pegomock.ReturnValue(nil))
	pegomock.When(jenkinsClient.GetJob(pegomock.EqString(testRepoName))).ThenReturn(pegomock.ReturnValue(job), pegomock.ReturnValue(nil))

	o := &deletecmd.DeleteApplicationOptions{
		CommonOptions: &commonOpts,
	}
	o.Args = []string{testRepoName}

	err = o.Run()
	assert.NoError(t, err)

	jenkinsClient.VerifyWasCalledOnce().DeleteJob(job)
}
