# Aegis

![Aegis](assets/aegis-icon.png "Aegis")

keep your secrets… secret

## Aegis Go SDK

Aegis SDK for Go language. 

This SDK enables workloads to directly communicate with Aegis Safe instead
of using a sidecar.

[Check out Aegis’ README][aegis-readme] for more information about the project, 
high level design, contributing guidelines, and code of conduct.

[aegis]: https://github.com/zerotohero-dev/aegis "Aegis"
[aegis-readme]: https://github.com/zerotohero-dev/aegis/blob/main/README.md "Aegis README"

## Usage Example

Here is a demo workload that uses the `Fetch()` API to retrieve secrets from 
**Aegis Safe**.

```go
package main

import (
	"fmt"
	"github.com/zerotohero-dev/aegis-sdk-go/sentry"
	"time"
)

func main() {
	for {
		// Fetch the secret bound to this workload
		// using Aegis Go SDK:
		data, err := sentry.Fetch()

		if err != nil {
			fmt.Println("Failed. Will retry…")
		} else {
			fmt.Println("secret: '", data, "'")
		}

		time.Sleep(5 * time.Second)
	}
}
```

Here follows a possible Deployment descriptor for such a workload. 

Check out [Aegis demo workload manifests][demos] for additional examples.

[demos]: https://github.com/zerotohero-dev/aegis/tree/main/install/k8s/demo-workload "Demo Workloads"

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: aegis-workload-demo
  namespace: default
automountServiceAccountToken: false
---
apiVersion: spire.spiffe.io/v1alpha1
kind: ClusterSPIFFEID
metadata:
  name: aegis-workload-demo
spec:
  spiffeIDTemplate: "spiffe://aegis.ist/workload/aegis-workload-demo"
  podSelector:
    matchLabels:
      app.kubernetes.io/name: aegis-workload-demo
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aegis-workload-demo
  namespace: default
  labels:
    app.kubernetes.io/name: aegis-workload-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: aegis-workload-demo
  template:
    metadata:
      labels:
        app.kubernetes.io/name: aegis-workload-demo
    spec:
      serviceAccountName: aegis-workload-demo
      containers:
        - name: main
          image: z2hdev/aegis-workload-demo-using-sdk:0.7.0
          volumeMounts:
          - name: spire-agent-socket
            mountPath: /spire-agent-socket
            readOnly: true
          env:
          - name: SPIFFE_ENDPOINT_SOCKET
            value: unix:///spire-agent-socket/agent.sock
      volumes:
      - name: spire-agent-socket
        hostPath:
          path: /run/spire/sockets
          type: Directory
 ``` 
