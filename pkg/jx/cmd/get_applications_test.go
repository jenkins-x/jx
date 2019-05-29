package cmd_test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	generatedv1 "github.com/jenkins-x/jx/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	utiltesting "k8s.io/client-go/util/testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestBuildGitUrl(t *testing.T) {
	sourceRepository := &v1.SourceRepository{
		Spec: v1.SourceRepositorySpec{
			Provider: "https://github.com",
			Org:      "my-org",
			Repo:     "my-repo",
		},
	}
	gitURL := cmd.BuildGitURL(sourceRepository)
	assert.Equal(t, "https://github.com/my-org/my-repo.git", gitURL, "The string should contain a vaild git URL")

	sourceRepository = &v1.SourceRepository{
		Spec: v1.SourceRepositorySpec{
			Provider: "https://invalid",
		},
	}
	gitURL = cmd.BuildGitURL(sourceRepository)
	assert.Equal(t, "None Found", gitURL, "The string should contain the default value")
}

func TestGetSourceRepositoryForApplication(t *testing.T) {
	testServer, _, _ := testServerEnv(t, 200)
	defer testServer.Close()

	restClient, err := restClient(testServer)
	if err != nil {
		t.Fatalf("Error in creating REST client: %s", err)
	}

	client := generatedv1.New(restClient)
	sourceRepositories := client.SourceRepositories("default")
	sourceRepository, err := cmd.GetSourceRepositoryForApplication("fake", sourceRepositories)

	assert.Nil(t, sourceRepository)
	assert.NotNil(t, err)
	assert.Error(t, err)
}

func testServerEnv(t *testing.T, statusCode int) (*httptest.Server, *utiltesting.FakeHandler, *metav1.Status) {
	status := &metav1.Status{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: fmt.Sprintf("%s", metav1.StatusSuccess)}
	expectedBody, _ := runtime.Encode(scheme.Codecs.LegacyCodec(v1.SchemeGroupVersion), status)
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   statusCode,
		ResponseBody: string(expectedBody),
		T:            t,
	}
	testServer := httptest.NewServer(&fakeHandler)
	return testServer, &fakeHandler, status
}

func restClient(testServer *httptest.Server) (*rest.RESTClient, error) {
	c, err := rest.RESTClientFor(&rest.Config{
		Host: testServer.URL,
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &v1.SchemeGroupVersion,
			NegotiatedSerializer: serializer.DirectCodecFactory{CodecFactory: scheme.Codecs},
		},
		Username: "user",
		Password: "pass",
	})
	return c, err
}
