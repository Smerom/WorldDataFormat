package worldDataFormat_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestWorldDataFormat(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "WorldDataFormat Suite")
}
