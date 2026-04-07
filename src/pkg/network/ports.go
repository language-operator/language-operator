/*
Copyright 2025 Langop Team.

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

package network

const (
	// GatewayContainerPort is the port the LiteLLM process listens on inside the pod.
	GatewayContainerPort = 4000
	// GatewayServicePort is the port exposed by the gateway Service (maps → GatewayContainerPort).
	GatewayServicePort = 8000

	// OTELGRPCPort is the standard OpenTelemetry gRPC receiver port.
	OTELGRPCPort = 4317
	// OTELHTTPPort is the standard OpenTelemetry HTTP receiver port.
	OTELHTTPPort = 4318

	// DNSPort is the standard DNS port (TCP and UDP).
	DNSPort = 53
)
