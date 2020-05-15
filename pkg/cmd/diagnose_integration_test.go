//// +build integration
//
package cmd_test

//
//import (
//	"os"
//	"testing"
//
//	"github.com/jenkins-x/jx/v2/pkg/cmd"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//
//	"github.com/jenkins-x/jx/v2/pkg/cmd/clients"
//	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
//	"github.com/jenkins-x/jx/v2/pkg/cmd/testhelpers"
//	"github.com/stretchr/testify/assert"
//
//	"github.com/jenkins-x/jx/v2/pkg/log"
//)
//
//const testDiagnoseNamespace = "jx-test"
//
//// Test_ExecuteDiagnose tests that jx diagnose works properly
//func Test_ExecuteDiagnose(t *testing.T) {
//	// Create home directory
//	origJXHome, testJXHome, err := testhelpers.CreateTestJxHomeDir()
//	assert.NoError(t, err, "failed to create a test JX Home directory")
//	defer func() {
//		err = testhelpers.CleanupTestJxHomeDir(origJXHome, testJXHome)
//		if err != nil {
//			log.Logger().Warnf("unable to remove temporary directory %s: %s", testJXHome, err)
//		}
//		log.Logger().Info("Deleted temporary jx home directory")
//	}()
//
//	// Test installing ksync locally
//	out := &testhelpers.FakeOut{}
//	commonOpts := opts.NewCommonOptionsWithTerm(clients.NewFactory(), os.Stdin, out, os.Stderr)
//
//	// Set batchmode to true for tests
//	commonOpts.BatchMode = true
//
//	// Get the current namespace, we will revert back to this after the tests are over
//	_, oldNs, _ := commonOpts.KubeClientAndNamespace()
//
//	// Create a dev namespace
//	commonOpts.SetDevNamespace(testDiagnoseNamespace)
//
//	command := cmd.NewCmdDiagnose(commonOpts)
//	err = command.Execute()
//	assert.NoError(t, err, "could not execute diagnose")
//
//	// Revert back to the old namespace
//	commonOpts.SetCurrentNamespace(oldNs)
//	client, err := commonOpts.KubeClient()
//	assert.NoError(t, err, "could not create jx client")
//
//	// Delete the namespace created for test purposes
//	client.CoreV1().Namespaces().Delete(testDiagnoseNamespace, &metav1.DeleteOptions{})
//	log.Logger().Infof("Deleting test namespace: %s", testDiagnoseNamespace)
//}
