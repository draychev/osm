package injector

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

var _ = Describe("Test Envoy configuration creation", func() {
	marshal := func(someStruct interface{}) string {
		jsonBytes, _ := json.MarshalIndent(someStruct, "-", "    ")
		return string(jsonBytes)
	}

	getExpectedXDSClusterStruct := func() map[string]interface{} {
		return map[string]interface{}{
			"name":                   "osm-controller",
			"connect_timeout":        "0.25s",
			"type":                   "LOGICAL_DNS",
			"http2_protocol_options": map[string]interface{}{},
			"transport_socket": map[string]interface{}{
				"name": "envoy.transport_sockets.tls",
				"typed_config": map[string]interface{}{
					"@type": "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
					"common_tls_context": map[string]interface{}{
						"alpn_protocols": []string{
							"h2",
						},
						"validation_context": map[string]interface{}{
							"trusted_ca": map[string]interface{}{
								"inline_bytes": "eHg=",
							},
						},
						"tls_params": map[string]interface{}{
							"tls_minimum_protocol_version": "TLSv1_2",
							"tls_maximum_protocol_version": "TLSv1_3",
						},
						"tls_certificates": []map[string]interface{}{
							{
								"certificate_chain": map[string]interface{}{
									"inline_bytes": "eHg=",
								},
								"private_key": map[string]interface{}{
									"inline_bytes": "eXk=",
								},
							},
						},
					},
				}},
			"load_assignment": map[string]interface{}{
				"endpoints": []map[string]interface{}{
					{
						"lb_endpoints": []map[string]interface{}{
							{
								"endpoint": map[string]interface{}{
									"address": map[string]interface{}{
										"socket_address": map[string]interface{}{
											"address":    "osm-controller.b.svc.cluster.local",
											"port_value": 15128,
										},
									},
								},
							},
						},
					},
				},
				"cluster_name": "osm-controller",
			},
		}
	}

	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	// This is the Bootstrap YAML generated for the Envoy proxies.
	// This is provisioned by the MutatingWebhook during the addition of a sidecar
	// to every new Pod that is being created in a namespace participating in the service mesh.
	// We deliberately leave this entire string literal here to document and visualize what the
	// generated YAML looks like!
	const expectedEnvoyConfig = `admin:
  access_log_path: /dev/stdout
  address:
    socket_address:
      address: 0.0.0.0
      port_value: "15000"
dynamic_resources:
  ads_config:
    api_type: GRPC
    grpc_services:
    - envoy_grpc:
        cluster_name: osm-controller
    set_node_on_first_message_only: true
    transport_api_version: V3
  cds_config:
    ads: {}
    resource_api_version: V3
  lds_config:
    ads: {}
    resource_api_version: V3
static_resources:
  clusters:
  - connect_timeout: 0.25s
    http2_protocol_options: {}
    load_assignment:
      cluster_name: osm-controller
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: osm-controller.b.svc.cluster.local
                port_value: 15128
    name: osm-controller
    transport_socket:
      name: envoy.transport_sockets.tls
      typed_config:
        '@type': type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
        common_tls_context:
          alpn_protocols:
          - h2
          tls_certificates:
          - certificate_chain:
              inline_bytes: eHg=
            private_key:
              inline_bytes: eXk=
          tls_params:
            tls_maximum_protocol_version: TLSv1_3
            tls_minimum_protocol_version: TLSv1_2
          validation_context:
            trusted_ca:
              inline_bytes: eHg=
    type: LOGICAL_DNS
  - connect_timeout: 0.25s
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: liveness_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 0.0.0.0
                port_value: 81
    name: liveness_cluster
    type: STATIC
  - connect_timeout: 0.25s
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: readiness_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 0.0.0.0
                port_value: 82
    name: readiness_cluster
    type: STATIC
  - connect_timeout: 0.25s
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: startup_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 0.0.0.0
                port_value: 83
    name: startup_cluster
    type: STATIC
  listeners:
  - address:
      socket_address:
        address: 0.0.0.0
        port_value: 15901
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
		  log_format:
			json_format:
			  authority: '%!R(MISSING)EQ(:AUTHORITY)%!'(MISSING)
			  bytes_received: '%!B(MISSING)YTES_RECEIVED%!'(MISSING)
			  bytes_sent: '%!B(MISSING)YTES_SENT%!'(MISSING)
			  duration: '%!D(MISSING)URATION%!'(MISSING)
			  method: '%!R(MISSING)EQ(:METHOD)%!'(MISSING)
			  path: '%!R(MISSING)EQ(X-ENVOY-ORIGINAL-PATH?:PATH)%!'(MISSING)
			  protocol: '%!P(MISSING)ROTOCOL%!'(MISSING)
			  request_id: '%!R(MISSING)EQ(X-REQUEST-ID)%!'(MISSING)
			  requested_server_name: '%!R(MISSING)EQUESTED_SERVER_NAME%!'(MISSING)
			  response_code: '%!R(MISSING)ESPONSE_CODE%!'(MISSING)
			  response_code_details: '%!R(MISSING)ESPONSE_CODE_DETAILS%!'(MISSING)
			  response_flags: '%!R(MISSING)ESPONSE_FLAGS%!'(MISSING)
			  start_time: '%!S(MISSING)TART_TIME%!'(MISSING)
			  time_to_first_byte: '%!R(MISSING)ESPONSE_DURATION%!'(MISSING)
			  upstream_cluster: '%!U(MISSING)PSTREAM_CLUSTER%!'(MISSING)
			  upstream_host: '%!U(MISSING)PSTREAM_HOST%!'(MISSING)
			  upstream_service_time: '%!R(MISSING)ESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%!'(MISSING)
			  user_agent: '%!R(MISSING)EQ(USER-AGENT)%!'(MISSING)
			  x_forwarded_for: '%!R(MISSING)EQ(X-FORWARDED-FOR)%!'(MISSING)
		  path: /dev/stdout
	  codec_type: AUTO
          http_filters:
          - name: envoy.filters.http.router
          route_config:
            name: local_route
            virtual_hosts:
            - domains:
              - '*'
              name: local_service
              routes:
              - match:
                  prefix: /osm-liveness-probe
                route:
                  cluster: liveness_cluster
                  prefix_rewrite: /liveness
          stat_prefix: health_probes_http
    name: liveness_listener
  - address:
      socket_address:
        address: 0.0.0.0
        port_value: 15902
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
		  log_format:
			json_format:
			  authority: '%!R(MISSING)EQ(:AUTHORITY)%!'(MISSING)
			  bytes_received: '%!B(MISSING)YTES_RECEIVED%!'(MISSING)
			  bytes_sent: '%!B(MISSING)YTES_SENT%!'(MISSING)
			  duration: '%!D(MISSING)URATION%!'(MISSING)
			  method: '%!R(MISSING)EQ(:METHOD)%!'(MISSING)
			  path: '%!R(MISSING)EQ(X-ENVOY-ORIGINAL-PATH?:PATH)%!'(MISSING)
			  protocol: '%!P(MISSING)ROTOCOL%!'(MISSING)
			  request_id: '%!R(MISSING)EQ(X-REQUEST-ID)%!'(MISSING)
			  requested_server_name: '%!R(MISSING)EQUESTED_SERVER_NAME%!'(MISSING)
			  response_code: '%!R(MISSING)ESPONSE_CODE%!'(MISSING)
			  response_code_details: '%!R(MISSING)ESPONSE_CODE_DETAILS%!'(MISSING)
			  response_flags: '%!R(MISSING)ESPONSE_FLAGS%!'(MISSING)
			  start_time: '%!S(MISSING)TART_TIME%!'(MISSING)
			  time_to_first_byte: '%!R(MISSING)ESPONSE_DURATION%!'(MISSING)
			  upstream_cluster: '%!U(MISSING)PSTREAM_CLUSTER%!'(MISSING)
			  upstream_host: '%!U(MISSING)PSTREAM_HOST%!'(MISSING)
			  upstream_service_time: '%!R(MISSING)ESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%!'(MISSING)
			  user_agent: '%!R(MISSING)EQ(USER-AGENT)%!'(MISSING)
			  x_forwarded_for: '%!R(MISSING)EQ(X-FORWARDED-FOR)%!'(MISSING)
		  path: /dev/stdout
	  codec_type: AUTO
          http_filters:
          - name: envoy.filters.http.router
          route_config:
            name: local_route
            virtual_hosts:
            - domains:
              - '*'
              name: local_service
              routes:
              - match:
                  prefix: /osm-readiness-probe
                route:
                  cluster: readiness_cluster
                  prefix_rewrite: /readiness
          stat_prefix: health_probes_http
    name: readiness_listener
  - address:
      socket_address:
        address: 0.0.0.0
        port_value: 15903
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
		  log_format:
			json_format:
			  authority: '%!R(MISSING)EQ(:AUTHORITY)%!'(MISSING)
			  bytes_received: '%!B(MISSING)YTES_RECEIVED%!'(MISSING)
			  bytes_sent: '%!B(MISSING)YTES_SENT%!'(MISSING)
			  duration: '%!D(MISSING)URATION%!'(MISSING)
			  method: '%!R(MISSING)EQ(:METHOD)%!'(MISSING)
			  path: '%!R(MISSING)EQ(X-ENVOY-ORIGINAL-PATH?:PATH)%!'(MISSING)
			  protocol: '%!P(MISSING)ROTOCOL%!'(MISSING)
			  request_id: '%!R(MISSING)EQ(X-REQUEST-ID)%!'(MISSING)
			  requested_server_name: '%!R(MISSING)EQUESTED_SERVER_NAME%!'(MISSING)
			  response_code: '%!R(MISSING)ESPONSE_CODE%!'(MISSING)
			  response_code_details: '%!R(MISSING)ESPONSE_CODE_DETAILS%!'(MISSING)
			  response_flags: '%!R(MISSING)ESPONSE_FLAGS%!'(MISSING)
			  start_time: '%!S(MISSING)TART_TIME%!'(MISSING)
			  time_to_first_byte: '%!R(MISSING)ESPONSE_DURATION%!'(MISSING)
			  upstream_cluster: '%!U(MISSING)PSTREAM_CLUSTER%!'(MISSING)
			  upstream_host: '%!U(MISSING)PSTREAM_HOST%!'(MISSING)
			  upstream_service_time: '%!R(MISSING)ESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%!'(MISSING)
			  user_agent: '%!R(MISSING)EQ(USER-AGENT)%!'(MISSING)
			  x_forwarded_for: '%!R(MISSING)EQ(X-FORWARDED-FOR)%!'(MISSING)
		  path: /dev/stdout
	  codec_type: AUTO
          http_filters:
          - name: envoy.filters.http.router
          route_config:
            name: local_route
            virtual_hosts:
            - domains:
              - '*'
              name: local_service
              routes:
              - match:
                  prefix: /osm-startup-probe
                route:
                  cluster: startup_cluster
                  prefix_rewrite: /startup
          stat_prefix: health_probes_http
    name: startup_listener
`

	cert := tresor.NewFakeCertificate()

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	config := envoyBootstrapConfigMeta{
		RootCert: base64.StdEncoding.EncodeToString(cert.GetIssuingCA()),
		Cert:     base64.StdEncoding.EncodeToString(cert.GetCertificateChain()),
		Key:      base64.StdEncoding.EncodeToString(cert.GetPrivateKey()),

		EnvoyAdminPort: 15000,

		XDSClusterName: "osm-controller",
		XDSHost:        "osm-controller.b.svc.cluster.local",
		XDSPort:        15128,
	}

	probes := healthProbes{
		liveness: &healthProbe{
			path: "/liveness",
			port: 81,
		},
		readiness: &healthProbe{
			path: "/readiness",
			port: 82,
		},
		startup: &healthProbe{
			path: "/startup",
			port: 83,
		},
	}

	Context("create envoy config", func() {
		It("creates envoy config", func() {
			actual, err := getEnvoyConfigYAML(config, mockConfigurator, probes)

			Expect(err).ToNot(HaveOccurred())

			Expect(string(actual)).To(Equal(expectedEnvoyConfig),
				fmt.Sprintf("Expected:\n%s\nActual:\n%s\n", expectedEnvoyConfig, string(actual)))
		})

		It("Creates bootstrap config for the Envoy proxy", func() {
			wh := &webhook{
				kubeClient:          fake.NewSimpleClientset(),
				kubeController:      k8s.NewMockController(gomock.NewController(GinkgoT())),
				nonInjectNamespaces: mapset.NewSet(),
			}
			name := uuid.New().String()
			namespace := "a"
			osmNamespace := "b"

			secret, err := wh.createEnvoyBootstrapConfig(name, namespace, osmNamespace, cert, probes)
			Expect(err).ToNot(HaveOccurred())

			expected := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					envoyBootstrapConfigFile: []byte(expectedEnvoyConfig),
				},
			}

			// Contains only the "bootstrap.yaml" key
			Expect(len(secret.Data)).To(Equal(1))

			Expect(secret.Data[envoyBootstrapConfigFile]).To(Equal(expected.Data[envoyBootstrapConfigFile]),
				fmt.Sprintf("Expected YAML: %s;\nActual YAML: %s\n", expected.Data, secret.Data))

			// Now check the entire struct
			Expect(*secret).To(Equal(expected))
		})
	})

	Context("Test getStaticResources()", func() {
		It("Creates static_resources Envoy struct", func() {
			actual := getXdsCluster(config, mockConfigurator, probes)
			expected := getExpectedXDSClusterStruct()

			expectedJSON := marshal(expected)
			actualJSON := marshal(actual)

			Expect(actualJSON).To(Equal(expectedJSON), fmt.Sprintf("Expected: %s\nActual struct: %s", expectedJSON, actualJSON))
		})
	})

	Context("Test getStaticResources()", func() {
		It("Creates static_resources Envoy struct", func() {
			actual := getStaticResources(config, mockConfigurator, probes)
			expected := map[string]interface{}{
				"clusters": []map[string]interface{}{
					getExpectedXDSClusterStruct(),
				},
			}

			expectedJSON := marshal(expected)
			actualJSON := marshal(actual)

			Expect(actualJSON).To(Equal(expectedJSON), fmt.Sprintf("Expected: %s\nActual struct: %s", expectedJSON, actualJSON))
		})
	})
})

// TODO(draychev): merge the two _
var _ = Describe("Test Envoy sidecar", func() {
	const (
		containerName = "-container-name-"
		envoyImage    = "-envoy-image-"
		nodeID        = "-node-id-"
		clusterID     = "-cluster-id-"
	)

	probes := healthProbes{
		liveness: &healthProbe{
			path: "/liveness",
			port: 81,
		},
		readiness: &healthProbe{
			path: "/readiness",
			port: 82,
		},
		startup: &healthProbe{
			path: "/startup",
			port: 83,
		},
	}

	mockCtrl := gomock.NewController(GinkgoT())
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	Context("create Envoy sidecar", func() {
		It("creates correct Envoy sidecar spec", func() {
			mockConfigurator.EXPECT().GetEnvoyLogLevel().Return("debug").Times(1)
			actual := getEnvoySidecarContainerSpec(containerName, envoyImage, nodeID, clusterID, mockConfigurator, probes)

			expected := corev1.Container{
				Name:            containerName,
				Image:           envoyImage,
				ImagePullPolicy: corev1.PullAlways,
				SecurityContext: &corev1.SecurityContext{
					RunAsUser: func() *int64 {
						uid := constants.EnvoyUID
						return &uid
					}(),
				},
				Ports: []corev1.ContainerPort{
					{
						Name:          constants.EnvoyAdminPortName,
						ContainerPort: constants.EnvoyAdminPort,
					},
					{
						Name:          constants.EnvoyInboundListenerPortName,
						ContainerPort: constants.EnvoyInboundListenerPort,
					},
					{
						Name:          constants.EnvoyInboundPrometheusListenerPortName,
						ContainerPort: constants.EnvoyPrometheusInboundListenerPort,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      envoyBootstrapConfigVolume,
						ReadOnly:  true,
						MountPath: envoyProxyConfigPath,
					},
				},
				Command: []string{
					"envoy",
				},
				Args: []string{
					"--log-level", "debug",
					"--config-path", "/etc/envoy/bootstrap.yaml",
					"--service-node", "$(POD_UID)/$(POD_NAMESPACE)/$(POD_IP)/$(SERVICE_ACCOUNT)/-node-id-",
					"--service-cluster", clusterID,
					"--bootstrap-version 3",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "POD_UID",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.uid",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_NAMESPACE",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.namespace",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_IP",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "status.podIP",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "SERVICE_ACCOUNT",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "spec.serviceAccountName",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
				},
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
