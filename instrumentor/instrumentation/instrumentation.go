package instrumentation

import (
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type CollectorInfo struct {
	Hostname string
	Port     int
}

func ModifyObject(original *v1.PodTemplateSpec, instrumentation *odigosv1.InstrumentedApplication, defaultSdks map[common.ProgrammingLanguage]common.OtelSdk) error {
	var modifiedContainers []v1.Container
	for _, container := range original.Spec.Containers {
		containerLanguage := getLanguageOfContainer(instrumentation, container.Name)
		if containerLanguage == nil {
			continue
		}

		otelSdk, found := defaultSdks[*containerLanguage]
		if !found {
			return fmt.Errorf("default sdk not found for language %s", *containerLanguage)
		}

		instrumentationDeviceName := common.InstrumentationDeviceName(*containerLanguage, otelSdk)

		if container.Resources.Limits == nil {
			container.Resources.Limits = make(map[v1.ResourceName]resource.Quantity)
		}
		container.Resources.Limits[v1.ResourceName(instrumentationDeviceName)] = resource.MustParse("1")

		modifiedContainers = append(modifiedContainers, container)
	}

	original.Spec.Containers = modifiedContainers
	return nil
}

func Revert(original *v1.PodTemplateSpec) {
	for _, container := range original.Spec.Containers {
		for resourceName := range container.Resources.Limits {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Limits, resourceName)
			}
		}
		// Is it needed?
		for resourceName := range container.Resources.Requests {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Requests, resourceName)
			}
		}
	}
}

func getLanguageOfContainer(instrumentation *odigosv1.InstrumentedApplication, containerName string) *common.ProgrammingLanguage {
	for _, l := range instrumentation.Spec.Languages {
		if l.ContainerName == containerName {
			return &l.Language
		}
	}

	return nil
}