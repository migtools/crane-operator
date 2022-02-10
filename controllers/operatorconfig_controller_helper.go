package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	errorutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

func getResources(path string) ([]string, error) {
	var data []byte

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(data), "---"), nil
}

func createResourcesFromFile(path string, ctx context.Context, log logr.Logger) error {

	var decoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	var dynamicResource dynamic.ResourceInterface
	errs := []error{}

	data, err := getResources(path)
	if err != nil {
		return err
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	dynamicConfig, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	for _, resource := range data {
		if len(resource) > 0 {
			obj := &unstructured.Unstructured{}
			_, gvk, err := decoder.Decode([]byte(resource), nil, obj)
			if err != nil {
				return err
			}

			mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				return err
			}

			if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
				// namespaced resources should specify the namespace
				dynamicResource = dynamicConfig.Resource(mapping.Resource).Namespace(InstallNamespace)
			} else {
				// for cluster-wide resources
				dynamicResource = dynamicConfig.Resource(mapping.Resource)
			}

			_, err = dynamicResource.Create(ctx, obj, metav1.CreateOptions{})

			if err != nil {
				if !errors.IsAlreadyExists(err) {
					errs = append(errs, err)
				} else {
					log.Info(fmt.Sprintf("%s already exists", obj.GetKind()), obj.GetName(), obj.GetNamespace())
				}
			} else {
				log.Info(fmt.Sprintf("%s created", obj.GetKind()), obj.GetName(), obj.GetNamespace())
			}
		}
	}

	if len(errs) != 0 {
		return errorutil.NewAggregate(errs)
	}

	return nil
}
