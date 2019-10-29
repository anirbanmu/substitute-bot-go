package replystorage

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestReplystorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "replystorage Suite")
}
