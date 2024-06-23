# Traefik Real IP Plugin

The `traefik_real_ip` plugin for Traefik enhances the ability to extract and set the real client IP address from the `X-Forwarded-For` header. This is particularly useful when Traefik is deployed behind a load balancer or proxy where the actual client IP address can be obscured.

## Features

- Extracts the real client IP address from the `X-Forwarded-For` header.
- Configurable depth (`forwardedForDepth`) for selecting which IP from `X-Forwarded-For` to use as the real IP.
- Sets the `X-Real-Ip` header with the determined real client IP address.
- Mitigates IP spoofing by ensuring the `X-Real-Ip` header reflects the actual client IP (`REAL_IP`).

## Configuration

### Configuration Options

The plugin supports the following configuration option:

| Option             | Description |
| ------------------ | ----------- |
| `forwardedForDepth`| Specifies the depth to look into the `X-Forwarded-For` header. Default is `1`, meaning it uses the last IP unless configured otherwise. |

### Example Configuration

Use it as a local plugin. And forwardedHeaders.insecure (proxy forward headers) property should be enabled.

### Deployment Configuration

#### Kubernetes Deployment Example

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    app: ingress-traefik-controller
  name: ingress-traefik-controller
  namespace: traefik
spec:
  selector:
    matchLabels:
      app: ingress-traefik-controller
  template:
    metadata:
      labels:
        app: ingress-traefik-controller
    spec:
      automountServiceAccountToken: true
      containers:
      - args:
        - --global.checknewversion
        - --global.sendanonymoususage
        - --entryPoints.metrics.address=:9100/tcp
        - --entryPoints.traefik.address=:9000/tcp
        - --entryPoints.web.proxyProtocol.insecure
        - --entryPoints.web.forwardedHeaders.insecure
        - --entryPoints.web.address=:8000/tcp
        - --entryPoints.websecure.address=:8443/tcp
        - --entryPoints.websecure.proxyProtocol.insecure
        - --entryPoints.websecure.forwardedHeaders.insecure
        - --api.dashboard=true
        - --ping=true
        - --metrics.prometheus=true
        - --metrics.prometheus.entrypoint=metrics
        - --providers.kubernetescrd
        - --providers.kubernetesingress
        - --entryPoints.websecure.http.tls=true
        - --log.level=INFO
        - --experimental.localPlugins.traefik-real-ip.modulename=github.com/r3d-shadow/traefik-real-ip
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
        image: docker.io/traefik:v3.0.2
        imagePullPolicy: IfNotPresent
        livenessProbe:
          failureThreshold: 3
          httpGet:
            path: /ping
            port: 9000
            scheme: HTTP
          initialDelaySeconds: 2
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 2
        name: ingress-traefik-controller
        ports:
        - containerPort: 9100
          name: metrics
          protocol: TCP
        - containerPort: 9000
          name: traefik
          protocol: TCP
        - containerPort: 8000
          name: web
          protocol: TCP
        - containerPort: 8443
          name: websecure
          protocol: TCP
        readinessProbe:
          failureThreshold: 1
          httpGet:
            path: /ping
            port: 9000
            scheme: HTTP
          initialDelaySeconds: 2
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 2
        resources:
          limits:
            cpu: 200m
            memory: 100Mi
          requests:
            cpu: 100m
            memory: 50Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
          runAsGroup: 65532
          runAsNonRoot: true
          runAsUser: 65532
        volumeMounts:
        - mountPath: /data
          name: data
        - mountPath: /tmp
          name: tmp
        - mountPath: /plugins-storage
          name: plugins-storage
        - mountPath: /plugins-local
          name: plugins-local
      initContainers:
      - args:
        - |
          apk update && apk add git && git clone https://github.com/r3d-shadow/traefik-real-ip /plugins-local/src/github.com/r3d-shadow/traefik-real-ip --depth 1 --single-branch --branch main
        command:
        - /bin/sh
        - -c
        image: alpine:3
        imagePullPolicy: IfNotPresent
        name: install-plugin
        resources: {}
        securityContext:
          runAsUser: 0
        volumeMounts:
        - mountPath: /plugins-local
          name: plugins-local
      restartPolicy: Always
      securityContext: {}
      serviceAccountName: ingress-traefik-controller
      terminationGracePeriodSeconds: 60
      volumes:
      - emptyDir: {}
        name: data
      - emptyDir: {}
        name: tmp
      - emptyDir: {}
        name: plugins-storage
      - emptyDir: {}
        name: plugins-local
  updateStrategy:
    rollingUpdate:
      maxSurge: 0
      maxUnavailable: 1
    type: RollingUpdate
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: whoami-ingressroute
  namespace: test
spec:
  entryPoints:
  - web
  - websecure
  routes:
  - kind: Rule
    match: Host(`example.com`) && PathPrefix(`/whoami`)
    middlewares:
    - name: traefik-real-ip
    services:
    - name: whoami
      port: 3000
---
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: traefik-real-ip
  namespace: test
spec:
  plugin:
    traefik-real-ip:
      forwardedForDepth: 2
```

### Configuration Documentation

This plugin ensures that Traefik accurately determines the real client IP address by evaluating the `X-Forwarded-For` header. Adjust the `forwardedForDepth` parameter to suit your environment and security requirements.

### Preventing IP Spoofing

To prevent IP spoofing attacks, configure the `forwardedForDepth` parameter appropriately. For instance, with `forwardedForDepth: 2`, the plugin ensures that the `X-Real-Ip` header always reflects the actual client IP (`REAL_IP`) from the `X-Forwarded-For` header.

### Example Usage

When sending a request with a custom `X-Forwarded-For` header:

```bash
curl -X POST https://example.com/whoami -H "X-Forwarded-For: SPOOF_IP, REAL_IP, PROXY_IP1, PROXY_IP2, PROXY_IP3"
```

Assuming `forwardedForDepth: 2`, the resulting headers would include:

```
Hostname: <hostname>
IP: <actual_real_ip>
...
X-Real-Ip: <actual_real_ip>
```

This demonstrates how the plugin accurately sets the `X-Real-Ip` header based on the `X-Forwarded-For` header, ensuring correct identification of the client IP even when Traefik is behind a proxy or load balancer or cdn.
