package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test patch functions", func() {
	Context("Test updateLabels", func() {
		envoyUID := "abc"
		It("adds", func() {
			pod := tests.NewPodTestFixture("ns", "pod-name")
			pod.Labels = nil
			actual := updateLabels(&pod, envoyUID)
			expected := &JSONPatchOperation{
				Op:    "add",
				Path:  "/metadata/labels",
				Value: map[string]string{"osm-envoy-uid": "abc"},
			}
			Expect(actual).To(Equal(expected))
		})

		It("replaces", func() {
			pod := tests.NewPodTestFixture("ns", "pod-name")
			actual := updateLabels(&pod, envoyUID)
			expected := &JSONPatchOperation{
				Op:    "replace",
				Path:  "/metadata/labels/osm-envoy-uid",
				Value: "abc",
			}
			Expect(actual).To(Equal(expected))
		})

		It("creates a list of patches for non-nil target", func() {
			target := map[string]string{
				"one": "1",
				"two": "2",
			}
			add := map[string]string{
				"three": "3",
				"two":   "2",
			}
			basePath := "/base/path/"
			actual := updateAnnotation(target, add, basePath)
			expected := []JSONPatchOperation{
				{
					Op:    "add",
					Path:  "/base/path/three",
					Value: "3",
				},
				{
					Op:    "replace",
					Path:  "/base/path/two",
					Value: "2",
				},
			}
			Expect(actual).To(Equal(expected))
		})

		It("creates a list of patches for nil target", func() {
			add := map[string]string{
				"three": "3",
				"two":   "2",
			}
			basePath := "/base/path/"

			actual := updateAnnotation(nil, add, basePath)

			two := JSONPatchOperation{

				Op:    "add",
				Path:  "/base/path/two",
				Value: "2",
			}
			Expect(actual).To(ContainElement(two))

			three := JSONPatchOperation{
				Op:    "add",
				Path:  "/base/path/three",
				Value: "3",
			}
			Expect(actual).To(ContainElement(three))
		})
	})
})
