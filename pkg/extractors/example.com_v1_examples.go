// Example extractor
package extractors

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	Register(schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1",
		Resource: "examples",
	}, example_com_v1_examples{})
}

type example_com_v1_examples struct{}

func (e example_com_v1_examples) GetFieldValue(obj *unstructured.Unstructured) (string, error) {
	return "", nil
}
func (e example_com_v1_examples) GetFieldTimestamp(obj *unstructured.Unstructured) (time.Time, error) {
	return time.Now(), nil
}
