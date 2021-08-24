package extractors

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	Register(schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachineinstances",
	}, kubevirt_io_v1_virtualmachineinstances{})
}

type kubevirt_io_v1_virtualmachineinstances struct{}

func (e kubevirt_io_v1_virtualmachineinstances) GetFieldValue(obj *unstructured.Unstructured) (string, error) {
	fields := []string{"status", "phase"}

	phase, found, err := unstructured.NestedString(obj.Object, fields...)
	if err != nil {
		return "", err
	}
	if !found {
		return "", ErrFieldsNotFound(fields...)
	}
	return phase, nil
}

func (e kubevirt_io_v1_virtualmachineinstances) GetFieldTimestamp(obj *unstructured.Unstructured) (time.Time, error) {
	fields := []string{"status", "phaseTransitionTimestamps"}

	fieldValue, err := e.GetFieldValue(obj)
	if err != nil {
		return time.Time{}, fmt.Errorf("extracting field timestamp: %w", err)
	}

	timestamps, found, err := unstructured.NestedSlice(obj.Object, fields...)
	if err != nil {
		return time.Time{}, err
	}
	if !found {
		return time.Time{}, ErrFieldsNotFound(fields...)
	}

	for _, timestamp := range timestamps {
		timestampMap := timestamp.(map[string]interface{})
		if timestampMap["phase"] == fieldValue {
			timestampValue, err := time.Parse(time.RFC3339, timestampMap["phaseTransitionTimestamp"].(string))
			if err != nil {
				return time.Time{}, err
			}
			return timestampValue, nil
		}
	}

	return time.Now(), nil
}
