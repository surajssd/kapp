/*
Copyright 2017 The Kedge Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spec

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api_v1 "k8s.io/client-go/pkg/api/v1"
	ext_v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	// install api (register and add types to api.Schema)
	_ "k8s.io/client-go/pkg/api/install"
	_ "k8s.io/client-go/pkg/apis/extensions/install"
)

// Fix

func fixServices(services []ServiceSpecMod, appName string) ([]ServiceSpecMod, error) {
	for i, service := range services {
		// auto populate service name if only one service is specified
		if service.Name == "" {
			if len(services) == 1 {
				service.Name = appName
			} else {
				return nil, errors.New("More than one service mentioned, please specify name for each one")
			}
		}

		// this should be the last statement in this for loop
		services[i] = service
	}
	return services, nil
}

func fixVolumeClaims(volumeClaims []VolumeClaim, appName string) ([]VolumeClaim, error) {
	for i, pVolume := range volumeClaims {
		if pVolume.Name == "" {
			if len(volumeClaims) == 1 {
				pVolume.Name = appName
			} else {
				return nil, errors.New("More than one persistent volume mentioned," +
					" please specify name for each one")
			}
		}
		volumeClaims[i] = pVolume
	}
	return volumeClaims, nil
}

func fixConfigMaps(configMaps []ConfigMapMod, appName string) ([]ConfigMapMod, error) {
	// if only one configMap is defined and its name is not specified
	if len(configMaps) == 1 && configMaps[0].Name == "" {
		configMaps[0].Name = appName
	} else if len(configMaps) > 1 {
		// if multiple configMaps is defined then each should have a name
		for cdn, cd := range configMaps {
			if cd.Name == "" {
				return nil, fmt.Errorf("name not specified for app.configMaps[%d]", cdn)
			}
		}
	}
	return configMaps, nil
}

func fixSecrets(secrets []SecretMod, appName string) ([]SecretMod, error) {
	// populate secret name only if one secret is specified
	if len(secrets) == 1 && secrets[0].Name == "" {
		secrets[0].Name = appName
	} else if len(secrets) > 1 {
		for i, sec := range secrets {
			if sec.Name == "" {
				return nil, fmt.Errorf("name not specified for app.secrets[%d]", i)
			}
		}
	}
	return secrets, nil
}

func fixContainers(containers []Container, appName string) ([]Container, error) {
	// if only one container set name of it as app name
	if len(containers) == 1 && containers[0].Name == "" {
		containers[0].Name = appName
	} else if len(containers) > 1 {
		// check if all the containers have a name
		// if not fail giving error
		for cn, c := range containers {
			if c.Name == "" {
				return nil, fmt.Errorf("app %q: container name not defined for app.containers[%d]", appName, cn)
			}
		}
	}
	return containers, nil
}

// Transform

func (app *ControllerFields) getLabels() map[string]string {
	labels := map[string]string{"app": app.Name}
	return labels
}

func (app *ControllerFields) createIngresses() ([]runtime.Object, error) {
	var ings []runtime.Object

	for _, i := range app.Ingresses {
		ing := &ext_v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:   i.Name,
				Labels: app.Labels,
			},
			Spec: i.IngressSpec,
		}
		ings = append(ings, ing)
	}
	return ings, nil
}

func (app *ControllerFields) createServices() ([]runtime.Object, error) {
	var svcs []runtime.Object
	for _, s := range app.Services {
		svc := &api_v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:   s.Name,
				Labels: app.Labels,
			},
			Spec: s.ServiceSpec,
		}
		for _, servicePortMod := range s.Ports {
			svc.Spec.Ports = append(svc.Spec.Ports, servicePortMod.ServicePort)
		}

		for _, portMapping := range s.PortMappings {
			servicePort, err := parsePortMapping(portMapping)
			if err != nil {
				return nil, errors.Wrap(err, "unable to parse port mapping")
			}
			svc.Spec.Ports = append(svc.Spec.Ports, *servicePort)
		}

		populateServicePortNames(svc.Name, svc.Spec.Ports)

		if len(svc.Spec.Selector) == 0 {
			svc.Spec.Selector = app.Labels
		}
		svcs = append(svcs, svc)

		// Generate ingress if "endpoint" is mentioned in app.Services.Ports[].Endpoint
		for _, port := range s.Ports {
			if port.Endpoint != "" {
				var host string
				var path string
				endpoint := strings.SplitN(port.Endpoint, "/", 2)
				switch len(endpoint) {
				case 1:
					host = endpoint[0]
					path = "/"
				case 2:
					host = endpoint[0]
					path = "/" + endpoint[1]
				default:
					return nil, fmt.Errorf("Invalid syntax for endpoint: %v", port.Endpoint)
				}

				ingressName := s.Name + "-" + strconv.FormatInt(int64(port.Port), 10)
				endpointIngress := &ext_v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:   ingressName,
						Labels: app.Labels,
					},
					Spec: ext_v1beta1.IngressSpec{
						Rules: []ext_v1beta1.IngressRule{
							{
								Host: host,
								IngressRuleValue: ext_v1beta1.IngressRuleValue{
									HTTP: &ext_v1beta1.HTTPIngressRuleValue{
										Paths: []ext_v1beta1.HTTPIngressPath{
											{
												Path: path,
												Backend: ext_v1beta1.IngressBackend{
													ServiceName: s.Name,
													ServicePort: intstr.IntOrString{
														IntVal: port.Port,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}
				svcs = append(svcs, endpointIngress)
			}
		}
	}
	return svcs, nil
}

// create PVC reading the root level persistent volume field
func (app *ControllerFields) createPVC() ([]runtime.Object, error) {
	var pvcs []runtime.Object
	for _, v := range app.VolumeClaims {
		// check for conditions where user has given both conflicting fields
		// or not given either fields
		if v.Size != "" && v.Resources.Requests != nil {
			return nil, fmt.Errorf("persistent volume %q, cannot provide size and resources at the same time", v.Name)
		}
		if v.Size == "" && v.Resources.Requests == nil {
			return nil, fmt.Errorf("persistent volume %q, please provide size or resources, none given", v.Name)
		}

		// if user has given size then create a "api_v1.ResourceRequirements"
		// because this can be fed to pvc directly
		if v.Size != "" {
			size, err := resource.ParseQuantity(v.Size)
			if err != nil {
				return nil, errors.Wrap(err, "could not read volume size")
			}
			// update the volume's resource so that it can be fed
			v.Resources = api_v1.ResourceRequirements{
				Requests: api_v1.ResourceList{
					api_v1.ResourceStorage: size,
				},
			}
		}
		// setting the default accessmode if none given by user
		if len(v.AccessModes) == 0 {
			v.AccessModes = []api_v1.PersistentVolumeAccessMode{api_v1.ReadWriteOnce}
		}
		pvcs = append(pvcs, &api_v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:   v.Name,
				Labels: app.Labels,
			},
			// since we updated the pvc spec before so this can be directly fed
			// without having to do any addition extra
			Spec: api_v1.PersistentVolumeClaimSpec(v.PersistentVolumeClaimSpec),
		})
	}
	return pvcs, nil
}

func (app *ControllerFields) createSecrets() ([]runtime.Object, error) {
	var secrets []runtime.Object

	for _, s := range app.Secrets {
		secret := &api_v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   s.Name,
				Labels: app.Labels,
			},
			Data:       s.Data,
			StringData: s.StringData,
			Type:       s.Type,
		}
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

// CreateK8sObjects, if given object DeploymentSpecMod, this function reads
// them and returns kubernetes objects as list of runtime.Object
// If the deployment is using field 'extraResources' then it will
// also return file names mentioned there as list of string
func (app *ControllerFields) CreateK8sObjects() ([]runtime.Object, []string, error) {
	var objects []runtime.Object

	if app.Labels == nil {
		app.Labels = app.getLabels()
	}

	svcs, err := app.createServices()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to create Kubernetes Service")
	}

	ings, err := app.createIngresses()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to create Kubernetes Ingresses")
	}

	secs, err := app.createSecrets()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Unable to create Kubernetes Secrets")
	}

	app.PodSpec.Containers, err = populateContainers(app.Containers, app.ConfigMaps, app.Secrets)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "deployment %q", app.Name)
	}
	log.Debugf("object after population: %#v\n", app)

	app.PodSpec.InitContainers, err = populateContainers(app.InitContainers, app.ConfigMaps, app.Secrets)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "deployment %q", app.Name)
	}
	log.Debugf("object after population: %#v\n", app)

	// create pvc for each root level persistent volume
	pvcs, err := app.createPVC()
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create Persistent Volume Claims")
	}

	vols, err := populateVolumes(app.PodSpec.Containers, app.VolumeClaims, app.PodSpec.Volumes)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "deployment %q", app.Name)
	}
	app.PodSpec.Volumes = append(app.PodSpec.Volumes, vols...)

	var configMap []runtime.Object
	for _, cd := range app.ConfigMaps {
		cm := &api_v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:   cd.Name,
				Labels: app.Labels,
			},
			Data: cd.Data,
		}

		configMap = append(configMap, cm)
	}

	// please keep the order of the artifacts addition as it is

	// adding non-controller objects
	objects = append(objects, pvcs...)
	log.Debugf("app: %s, pvc: %s\n", app.Name, spew.Sprint(pvcs))

	objects = append(objects, svcs...)
	log.Debugf("app: %s, service: %s\n", app.Name, spew.Sprint(svcs))

	objects = append(objects, ings...)
	log.Debugf("app: %s, ingress: %s\n", app.Name, spew.Sprint(ings))

	objects = append(objects, secs...)
	log.Debugf("app: %s, secret: %s\n", app.Name, spew.Sprint(secs))

	objects = append(objects, configMap...)
	log.Debugf("app: %s, configMap: %s\n", app.Name, spew.Sprint(configMap))

	return objects, app.ExtraResources, nil
}

// Validate

func validateVolumeClaims(vcs []VolumeClaim) error {
	// find the duplicate volume claim names, if found any then error out
	vcmap := make(map[string]interface{})
	for _, vc := range vcs {
		if _, ok := vcmap[vc.Name]; !ok {
			// value here does not matter
			vcmap[vc.Name] = nil
		} else {
			return fmt.Errorf("duplicate entry of volume claim %q", vc.Name)
		}
	}
	return nil
}
