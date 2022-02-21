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
	s.Setup()
}

func (s *integrationTestSuite) TearDownSuite() {
	s.Teardown()
}

func TestIntegration(t *testing.T) {
	suite.Run(t, new(integrationTestSuite))
}
