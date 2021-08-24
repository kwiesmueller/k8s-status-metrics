package collector

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kwiesmueller/k8s-status-metrics/pkg/extractors"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

type FieldRecorder struct {
	gvr        schema.GroupVersionResource
	recordings map[namespacedName]record
	// finalPhase defines at which phase a resource is considered done
	// and may be removed.
	finalPhase string

	decoder   runtime.Decoder
	extractor extractors.Extractor

	histogram *prometheus.GaugeVec
}

func NewFieldRecorder(gvr schema.GroupVersionResource, finalPhase string) (FieldRecorder, error) {
	recorder := FieldRecorder{
		gvr:        gvr,
		recordings: make(map[namespacedName]record),
		finalPhase: finalPhase,
		decoder:    yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme),
		histogram: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: normalizeMetric(fmt.Sprintf("resource_phase_duration_%s_%s_%s_seconds", gvr.Group, gvr.Version, gvr.Resource)),
			Help: "Gauge for the time spent in resource phases.",
		}, []string{"namespace", "name", "phase"}),
	}

	err := prometheus.Register(recorder.histogram)
	if err != nil {
		return FieldRecorder{}, err
	}

	recorder.extractor, err = extractors.Get(gvr)
	if err != nil {
		return FieldRecorder{}, err
	}

	return recorder, nil
}

type namespacedName struct {
	namespace string
	name      string
}

func (n namespacedName) String() string {
	return n.namespace + "/" + n.name
}

type record struct {
	timestamp time.Time
	phase     string
}

func (r FieldRecorder) RecordObject(obj runtime.Object) error {
	uObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return errors.New("obj is not unstructured")
	}

	key := namespacedName{
		namespace: uObj.GetNamespace(),
		name:      uObj.GetName(),
	}

	phase, err := r.extractor.GetFieldValue(uObj)
	if err != nil {
		return err
	}
	phaseTimestamp, err := r.extractor.GetFieldTimestamp(uObj)
	if err != nil {
		return err
	}

	existingRecord, exists := r.recordings[key]
	if exists && existingRecord.phase != phase {
		fmt.Println(time.Now().Format(time.RFC3339), key.String(), phase)
		r.recordTransition(key,
			existingRecord.phase,
			existingRecord.timestamp,
			phaseTimestamp,
		)
	}

	if phase == r.finalPhase {
		delete(r.recordings, key)
		return nil
	}

	r.recordings[key] = record{
		phase:     phase,
		timestamp: phaseTimestamp,
	}

	return nil
}

func (r FieldRecorder) DeleteObject(obj runtime.Object) error {
	uObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return errors.New("obj is not unstructured")
	}

	key := namespacedName{
		namespace: uObj.GetNamespace(),
		name:      uObj.GetName(),
	}

	phaseTimestamp, err := r.extractor.GetFieldTimestamp(uObj)
	if err != nil {
		return err
	}

	existingRecord, exists := r.recordings[key]
	if !exists {
		fmt.Println(time.Now().Format(time.RFC3339), key.String(), "Deleted")
		r.recordTransition(key,
			existingRecord.phase,
			existingRecord.timestamp,
			phaseTimestamp,
		)
	}

	delete(r.recordings, key)
	return nil
}

func (r FieldRecorder) recordTransition(key namespacedName, phase string, since, until time.Time) {
	r.histogram.WithLabelValues(key.namespace, key.name, phase).Set(until.Sub(since).Seconds())
}

func normalizeMetric(in string) string {
	return strings.ReplaceAll(strings.ReplaceAll(in, ".", "_"), "-", "_")
}
