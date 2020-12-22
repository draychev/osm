package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
)

var _ = Describe("Test functions configuring Envoy's payload health probe", func() {
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

	Context("test getVirtualHosts()", func() {
		It("creates Envoy config", func() {
			newPath := uuid.New().String()
			clusterName := uuid.New().String()
			originalProbePath := uuid.New().String()
			actual := getVirtualHosts(newPath, clusterName, originalProbePath)

			Expect(len(actual)).To(Equal(1))
			expected := map[string]interface{}{
				"name": "local_service",
				"domains": []string{
					"*",
				},
				"routes": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"prefix": newPath,
						},
						"route": map[string]interface{}{
							"cluster":        clusterName,
							"prefix_rewrite": originalProbePath,
						},
					},
				},
			}
			Expect(actual[0]).To(Equal(expected))
		})
	})

	Context("test getLivenessListener()", func() {
		It("creates Envoy config", func() {
			listenerName := "listener-name"
			clusterName := "cluster-name"
			newPath := "-new-path-"
			port := int32(12345)
			actual := getProbeListener(listenerName, clusterName, newPath, port, probes.liveness)
			expected := map[string]interface{}{
				"name": listenerName,
				"address": map[string]interface{}{
					"socket_address": map[string]interface{}{
						"address":    "0.0.0.0",
						"port_value": port,
					},
				},
				"filter_chains": []map[string]interface{}{
					{
						"filters": []map[string]interface{}{
							{
								"name": "envoy.filters.network.http_connection_manager",
								"typed_config": map[string]interface{}{
									"@type":       "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
									"stat_prefix": "health_probes_http",
									"access_log":  getAccessLog(),
									"codec_type":  "AUTO",
									"route_config": map[string]interface{}{
										"name":          "local_route",
										"virtual_hosts": getVirtualHosts(newPath, clusterName, probes.liveness.path),
									},
									"http_filters": []map[string]interface{}{
										{
											"name": "envoy.filters.http.router",
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("test getLivenessListener()", func() {
		It("creates Envoy config", func() {
			actual := getLivenessListener(probes.liveness)
			Expect(len(actual)).To(Equal(3))
		})
	})

	Context("test getReadinessListener()", func() {
		It("creates Envoy config", func() {
			actual := getReadinessListener(probes.readiness)
			Expect(len(actual)).To(Equal(3))
		})
	})

	Context("test getStartupListener()", func() {
		It("creates Envoy config", func() {
			actual := getStartupListener(probes.startup)
			Expect(len(actual)).To(Equal(3))
		})
	})

	Context("test getAccessLog()", func() {
		It("creates envoy config", func() {
			actual := getAccessLog()
			expected := []map[string]interface{}{
				{
					"typed_config": map[string]interface{}{
						"@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
						"path":  "/dev/stdout",
						"log_format": map[string]interface{}{
							"json_format": map[string]interface{}{
								"requested_server_name": "%REQUESTED_SERVER_NAME%",
								"response_flags":        "%RESPONSE_FLAGS%",
								"path":                  "%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%",
								"authority":             "%REQ(:AUTHORITY)%",
								"protocol":              "%PROTOCOL%",
								"response_code_details": "%RESPONSE_CODE_DETAILS%",
								"start_time":            "%START_TIME%",
								"upstream_host":         "%UPSTREAM_HOST%",
								"bytes_sent":            "%BYTES_SENT%",
								"upstream_service_time": "%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%",
								"upstream_cluster":      "%UPSTREAM_CLUSTER%",
								"response_code":         "%RESPONSE_CODE%",
								"duration":              "%DURATION%",
								"x_forwarded_for":       "%REQ(X-FORWARDED-FOR)%",
								"user_agent":            "%REQ(USER-AGENT)%",
								"method":                "%REQ(:METHOD)%",
								"time_to_first_byte":    "%RESPONSE_DURATION%",
								"bytes_received":        "%BYTES_RECEIVED%",
								"request_id":            "%REQ(X-REQUEST-ID)%",
							},
						},
					},
					"name": "envoy.access_loggers.file",
				},
			}
			Expect(actual).To(Equal(expected))
		})
	})
})
