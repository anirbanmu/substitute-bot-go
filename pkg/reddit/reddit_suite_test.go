package reddit

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestReddit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reddit Suite")
}
