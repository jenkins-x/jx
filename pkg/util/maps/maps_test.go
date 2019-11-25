package maps_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/util/maps"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Person struct {
	Name     string    `json:"name"`
	ID       string    `json:"id"`
	Embedded *Embedded `json:"embedded,omitempty"`
}

type Embedded struct {
	Greeting string `json:"greeting"`
}

func TestMaps(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Maps Test Suite")
}

var _ = Describe("Map utility functions", func() {

	Describe("#KeyValuesToMap", func() {
		It("should extract valid key value pairs from string array", func() {
			m := maps.KeyValuesToMap([]string{"foo=bar", "whatnot=cheese"})
			Expect(m).ShouldNot(BeNil(), "util.KeysToMap() returned nil")
			Expect(m["foo"]).Should(Equal("bar"), "map does not contain foo=bar")
			Expect(m["whatnot"]).Should(Equal("cheese"), "map does not contain whatnot=cheese")
		})
	})

	Describe("#MapToKeyValues", func() {
		It("should create key/value string array from map", func() {
			values := maps.MapToKeyValues(map[string]string{
				"foo":     "bar",
				"whatnot": "cheese",
			})
			Expect(values).Should(Equal([]string{"foo=bar", "whatnot=cheese"}))
		})
	})

	Describe("#ToObjectMap", func() {
		It("should convert flat struct to object map", func() {
			john := &Person{
				Name: "John Doe",
				ID:   "42",
			}
			objectMap, err := maps.ToObjectMap(john)
			Expect(err).Should(BeNil())
			Expect(objectMap).Should(HaveKeyWithValue("name", "John Doe"))
			Expect(objectMap).Should(HaveKeyWithValue("id", "42"))
			Expect(objectMap).ShouldNot(HaveKey("embedded"))
		})

		It("should convert nested struct to object map", func() {
			greet := &Embedded{
				Greeting: "Mister",
			}
			john := &Person{
				Name:     "John Doe",
				ID:       "42",
				Embedded: greet,
			}
			objectMap, err := maps.ToObjectMap(john)
			Expect(err).Should(BeNil())
			Expect(objectMap).Should(HaveKeyWithValue("embedded", map[string]interface{}{"greeting": "Mister"}))
		})
	})
})
