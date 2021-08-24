# Kubernetes Status Metrics
A prototype for a tool to export metrics for status/field transitions of Kubernetes resources.

## Usage

**Note: This tool is merely a protoype/proof-of-concept and does not satisfy any expectations for usability, documentation and testing yet.**

To use the tool run it with your inteded parameters.
For example:

```sh
go run cmd/collector/main.go --group=kubevirt.io --version=v1 --resource=virtualmachineinstances -l vm.kubevirt.io/name=probe-test
```

## Working with Resources

To make this work with any resource, you'll need to create a custom extractor. An example can be found in [pkg/extractors/example.com_v1_examples.go](pkg/extractors/example.com_v1_examples.go).

Extractors specify how to get the phase and timestamp from a resource.
This method is being used until there is time to use something like JSON Pointers or FieldPaths for a more dynamic extraction.

## Metrics
The tool exports metrics for every resource matching its config.
For example VMIs:

```
# HELP resource_phase_duration_kubevirt_io_v1_virtualmachineinstances_seconds Gauge for the time spent in resource phases.
# TYPE resource_phase_duration_kubevirt_io_v1_virtualmachineinstances_seconds gauge
resource_phase_duration_kubevirt_io_v1_virtualmachineinstances_seconds{name="probe-test",namespace="default",phase=""} 9.223372036854776e+09
resource_phase_duration_kubevirt_io_v1_virtualmachineinstances_seconds{name="probe-test",namespace="default",phase="Pending"} 1
resource_phase_duration_kubevirt_io_v1_virtualmachineinstances_seconds{name="probe-test",namespace="default",phase="Running"} 37
resource_phase_duration_kubevirt_io_v1_virtualmachineinstances_seconds{name="probe-test",namespace="default",phase="Scheduled"} 2
resource_phase_duration_kubevirt_io_v1_virtualmachineinstances_seconds{name="probe-test",namespace="default",phase="Scheduling"} 6
```