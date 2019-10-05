package reddit

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestReddit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reddit Suite")
}
