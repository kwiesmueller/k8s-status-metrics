package collector

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type App struct {
	group    *string
	version  *string
	resource *string
	kind     *string

	namespace     *string
	labelSelector *string
	fieldSelector *string
	finalPhase    *string

	metricsPort *int

	gvr           schema.GroupVersionResource
	clientFactory cmdutil.Factory
}

const (
	paramGroup         = "group"
	paramVersion       = "version"
	paramResource      = "resource"
	paramKind          = "kind"
	paramLabelSelector = "selector"
	paramFieldSelector = "field-selector"
	paramFinalPhase    = "final-phase"
	paramMetricsPort   = "metrics-port"
)

func NewCLIApp() App {
	app := App{}

	flags := pflag.NewFlagSet("Collector", pflag.ExitOnError)

	app.group = flags.String(paramGroup, "", "API group of the resource to watch")
	app.version = flags.String(paramVersion, "", "API version of the resource to watch")
	app.resource = flags.String(paramResource, "", "API resource to watch")
	app.kind = flags.String(paramKind, "", "API kind to watch")

	// based on https://github.com/kubernetes/kubectl/blob/master/pkg/cmd/get/get.go#L190
	app.labelSelector = flags.StringP(paramLabelSelector, "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	app.fieldSelector = flags.String(paramFieldSelector, "", "Selector (field query) to filter on, supports '=', '==', and '!='.(e.g. --field-selector key1=value1,key2=value2). The server only supports a limited number of field queries per type.")
	app.finalPhase = flags.String(paramFinalPhase, "Succeeded", "Final phase a resource will enter. Causes the timers for the resource to reset.")

	app.metricsPort = flags.Int(paramMetricsPort, 8080, "Port prometheus metrics get exposed at.")

	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.AddFlags(flags)
	app.clientFactory = cmdutil.NewFactory(kubeConfigFlags)

	app.namespace = kubeConfigFlags.Namespace

	flags.SortFlags = false
	flags.Parse(os.Args)

	if len(*app.resource) > 0 {
		app.gvr = schema.GroupVersionResource{
			Group:    *app.group,
			Version:  *app.version,
			Resource: *app.resource,
		}
	}

	return app
}

var errMissingParam = errors.New("missing parameter")

func (app App) Validate() error {
	if len(*app.version) < 1 {
		return fmt.Errorf("%w: %s", errMissingParam, paramVersion)
	}
	if len(*app.resource) < 1 && len(*app.kind) < 1 {
		return fmt.Errorf("%w: %s or %s", errMissingParam, paramResource, paramKind)
	}
	if len(*app.resource) > 0 && len(*app.kind) > 0 {
		return fmt.Errorf("invalid params: only one of %s or %s can be set", paramResource, paramKind)
	}

	return nil
}

func (app App) Execute() error {
	ctx := context.Background()

	if err := app.Validate(); err != nil {
		return err
	}

	if app.gvr.Empty() {
		discoveryClient, err := app.clientFactory.ToDiscoveryClient()
		if err != nil {
			return err
		}
		mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
		mapping, err := mapper.RESTMapping(app.GroupVersionKind().GroupKind(), app.GroupVersionKind().Version)
		if err != nil {
			return err
		}
		app.gvr = mapping.Resource
	}

	dynamicClient, err := app.clientFactory.DynamicClient()
	if err != nil {
		return err
	}

	resource := dynamicClient.Resource(app.gvr)

	listOptions := metav1.ListOptions{
		LabelSelector: strings.TrimSpace(*app.labelSelector),
		FieldSelector: strings.TrimSpace(*app.fieldSelector),
	}

	if len(*app.namespace) > 0 {
		fmt.Printf("watching %s in %s\n", app.gvr, *app.namespace)
	} else {
		fmt.Printf("watching %s\n", app.gvr)
	}

	watcher, err := resource.Namespace(*app.namespace).Watch(ctx, listOptions)
	if err != nil {
		return err
	}

	go app.ServeMetrics(watcher)

	return app.RunWatcher(watcher)
}

func (app App) RunWatcher(watcher watch.Interface) error {
	recorder, err := NewFieldRecorder(app.gvr, *app.finalPhase)
	if err != nil {
		return err
	}

	for event := range watcher.ResultChan() {
		if event.Type == watch.Deleted {
			if err := recorder.DeleteObject(event.Object); err != nil {
				fmt.Printf("recording error: %v\n", err)
			}
			continue
		}
		if event.Type != watch.Added && event.Type != watch.Modified {
			continue
		}
		if err := recorder.RecordObject(event.Object); err != nil {
			fmt.Printf("recording error: %v\n", err)
		}
	}
	return nil
}

func (app App) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   *app.group,
		Version: *app.version,
		Kind:    *app.kind,
	}
}

func (app App) ServeMetrics(watcher watch.Interface) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	fmt.Printf("exporting /metrics at port %d\n", *app.metricsPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *app.metricsPort), mux); err != nil {
		fmt.Println(err)
		watcher.Stop()
	}
}
