package github

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *UnitTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.RegisterActivity(&ReleaseProcess{
		Client:     &MockClient{},
		Repo:       &MockRepo{},
		BaseBranch: "master",
	})

	s.env.RegisterWorkflow(OrchestrateReleaseProcess)
	s.env.RegisterWorkflow(WaitUntilPRIsAccepted)
}

func (s *UnitTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}
func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func (s *UnitTestSuite) Test_SimpleWorkflow_Success() {
	s.env.ExecuteWorkflow(OrchestrateReleaseProcess, "1.33.7")

	s.env.RegisterDelayedCallback(
		func() {
			s.env.SignalWorkflow(ApproveSignal, nil)
		},
		time.Second*10,
	)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}
