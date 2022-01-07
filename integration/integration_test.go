package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/controller-runtime/pkg/client"

	referenceaddonv1alpha1 "github.com/openshift/reference-addon/apis/reference/v1alpha1"
	"github.com/openshift/reference-addon/integration"
)

type integrationTestSuite struct {
	suite.Suite
}

func (s *integrationTestSuite) SetupSuite() {
	// TODO(ykukreja): uncomment the following line once the support of setup/teardown of local registry is established
	//s.Setup()
}

func (s *integrationTestSuite) TearDownSuite() {
	// TODO(ykukreja): uncomment the following line once the support of setup/teardown of local registry is established
	//s.Teardown()
}

func (s *integrationTestSuite) referenceAddonCleanup(addon *referenceaddonv1alpha1.ReferenceAddon,
	ctx context.Context) {
	s.T().Logf("waiting for addon %s to be deleted", addon.Name)

	// delete Addon
	err := integration.Client.Delete(ctx, addon, client.PropagationPolicy("Foreground"))
	s.Require().NoError(client.IgnoreNotFound(err), "delete Addon: %v", addon)

	// wait until Addon is gone
	defaultAddonDeletionTimeout := 4 * time.Minute
	err = integration.WaitToBeGone(s.T(), defaultAddonDeletionTimeout, addon)
	s.Require().NoError(err, "wait for Addon to be deleted")
}

func TestIntegration(t *testing.T) {
	suite.Run(t, new(integrationTestSuite))
}
