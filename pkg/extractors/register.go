package extractors

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var registered map[schema.GroupVersionResource]Extractor = make(map[schema.GroupVersionResource]Extractor)

// Extractor defines the interface required to provide field extractors compatible with the collector package.
// To use custom resources with this project, add your extractor to this package in a dedicate file called
// %group%_%version%_%resource%.go with the values respectively replaced to match your resource.
// Your file should contain an init function that calls Register to become available.
type Extractor interface {
	GetFieldValue(obj *unstructured.Unstructured) (string, error)
	GetFieldTimestamp(obj *unstructured.Unstructured) (time.Time, error)
}

// Register an extractor for a GVR. If a extractor already exists, this will panic.
func Register(gvr schema.GroupVersionResource, extractor Extractor) {
	if _, exists := registered[gvr]; exists {
		panic("extractor already registeded for " + gvr.String())
	}
	registered[gvr] = extractor
}

// Get the extractor registered for GVR or return an error
func Get(gvr schema.GroupVersionResource) (Extractor, error) {
	extractor, found := registered[gvr]
	if !found {
		return nil, ErrExtractorNotFound(gvr)
	}
	return extractor, nil
}

const (
	errFieldsNotFound    = "field not found"
	errExtractorNotFound = "extractor not found"
)

func ErrFieldsNotFound(fields ...string) error {
	return fmt.Errorf("%s: %s", errFieldsNotFound, fields)
}

type ErrExtractorNotFound schema.GroupVersionResource

func (e ErrExtractorNotFound) Error() string {
	return fmt.Sprintf("%s for: %s", errExtractorNotFound, schema.GroupVersionResource(e))
}
