package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestingSuite struct {
	*require.Assertions
	proxy suite.Suite
}

func (suite *TestingSuite) T() *testing.T {
	return suite.proxy.T()
}

func (suite *TestingSuite) SetT(t *testing.T) {
	suite.proxy.SetT(t)
	suite.Assertions = require.New(t)
}

func (suite *TestingSuite) Require() *require.Assertions {
	return suite.proxy.Require()
}

func (suite *TestingSuite) Assert() *assert.Assertions {
	return suite.proxy.Assert()
}

func (suite *TestingSuite) Fn() *MockFunc {
	return NewMockFunc(suite.T())
}
