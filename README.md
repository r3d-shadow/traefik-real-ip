# Enhancing IP Transparency with Traefik and the `traefik_real_ip` Plugin

[Github Repo Link for traefik-real-ip Plugin](https://github.com/r3d-shadow/traefik-real-ip/)

For an insightful deep dive into how the X-Forwarded-For header impacts rate limiting and why setting the real client IP is essential, consider reading this [blog](https://red-shadow.live/blogs/bypassing-rate-limiting-using-the-x-forwarded-for-header-a-deep-dive). Understanding these concepts is pivotal in ensuring robust and secure application deployments.

## Introduction to Traefik

Traefik is a modern, dynamic reverse proxy and load balancer designed to simplify the deployment and management of applications. It's especially powerful in a Kubernetes environment, where it acts as an ingress controller, managing external access to the services running within the cluster.

### Key Features of Traefik:
- **Automatic Service Discovery**: Traefik dynamically discovers services in your infrastructure and automatically reconfigures itself.
- **Load Balancing**: Distributes incoming traffic across multiple service instances to ensure availability and reliability.
- **SSL Termination**: Automatically handles SSL certificates, allowing for secure communication.
- **Middleware Support**: Easily extend Traefik’s capabilities with middlewares for authentication, rate limiting, etc.

## Understanding Reverse Proxy and Kubernetes Ingress

A reverse proxy serves as an intermediary for requests from clients seeking resources from servers. Traefik, when configured as a reverse proxy, forwards client requests to the appropriate backend services based on the defined rules.

In Kubernetes, an **Ingress Controller** like Traefik manages the ingress traffic by defining rules for routing external traffic to internal services. This is crucial for enabling access to the applications running inside the cluster from the outside world.

## Deploying Custom Plugins in Traefik

One of the standout features of Traefik is its support for custom plugins. This capability allows users to extend Traefik’s functionality according to their specific needs. Plugins can be integrated as part of Traefik's pipeline, handling requests before they reach the backend services.

### Setting Up a Custom Local Plugin with Init Containers

To deploy a custom local plugin, we use an init container to download and set up the plugin source code in the correct directory. This approach is especially useful when you need to host the plugin locally within the Kubernetes cluster.

Here’s an example of how you can set up a custom local plugin with Traefik:

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
        - --log.level=DEBUG
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
```

### Understanding the Init Container

In the example above:
- **Init Container**: `install-plugin` is an init container that runs before the main Traefik container starts. It installs `git` and clones the plugin repository into the `/plugins-local` directory.
- **Volume Mounts**: The cloned plugin code is mounted into the main Traefik container at the `/plugins-local` path.

## The `traefik_real_ip` Plugin

The `traefik_real_ip` plugin for Traefik enhances the ability to accurately identify and set the real client IP address when Traefik is deployed behind multiple layers of proxies or load balancers.

### Why Real IP Matters?

In scenarios where Traefik sits behind a series of proxies (e.g., CDN, load balancers), the client’s original IP address can be obscured. Correctly identifying the client’s real IP address is crucial for:
- **Security**: Accurate IP logging helps in security audits and incident response.
- **Rate Limiting and Geo-blocking**: Applying policies based on the client’s real IP address.
- **Custom Application Logic**: Some applications might rely on the client’s IP for personalized content delivery or access control.

### How It Works

The plugin examines the `X-Forwarded-For` header, which carries a list of IP addresses representing the client’s route through proxies. By configuring the `forwardedForDepth` parameter, you specify how far back in the list to look to determine the real client IP.

### Configuration Options

- **`forwardedForDepth`**: Determines the position in the `X-Forwarded-For` header to use as the real IP. A value of `1` (default) means the last IP in the list is used. Increasing this value allows you to step back through the proxies to find the actual client IP.

### Preventing IP Spoofing

To prevent IP spoofing attacks, it’s important to configure the `forwardedForDepth` correctly. For example, with `forwardedForDepth: 2`, the plugin will pick the second last IP in the `X-Forwarded-For` header chain as the real IP, mitigating attempts to spoof the client's IP address.

### Example Usage

Let's illustrate the usage of the `traefik_real_ip` plugin with a practical example involving an `X-Forwarded-For` header:

Consider a scenario where a request is made with the following `X-Forwarded-For` header:

```
X-Forwarded-For: SPOOF_IP, REAL_IP, PROXY1, PROXY0
```

To simulate this scenario using curl:

```bash
curl -X POST https://example.com/whoami -H "X-Forwarded-For: SPOOF_IP"
```

In this setup:
- **X-Forwarded-For Header**: The header includes a chain of IP addresses, potentially including spoofed IPs.
- **Plugin Behavior**: With `forwardedForDepth: 2` configured in the `traefik_real_ip` plugin, it will set the `X-Real-Ip` header to `REAL_IP`, which is the second IP in the `X-Forwarded-For` chain after the spoofed IP.
- **X-Real-Ip Header**: After processing, the X-Real-Ip will accurately reflect the real client IP as follows:

```
X-Real-Ip: REAL_IP
```

This ensures that the application receives the correct client IP information despite the presence of spoofed IPs in the `X-Forwarded-For` header, thereby enhancing security and trustworthiness in identifying client origins.

### Full Configuration Example

Here’s a configuration setup for deploying the `traefik_real_ip` plugin in Kubernetes with Traefik:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: whoami
  name: whoami
spec:
  selector:
    matchLabels:
      app: whoami
  template:
    metadata:
      labels:
        app: whoami
    spec:
      containers:
      - image: containous/whoami
        imagePullPolicy: Always
        name: whoami
        ports:
        - containerPort: 80
          name: web
          protocol: TCP
        resources: {}
      restartPolicy: Always
---
apiVersion: v1
kind: Service
metadata:
  name: whoami
spec:
  ports:
  - name: web
    port: 80
    protocol: TCP
    targetPort: web
  selector:
    app: whoami
  type: ClusterIP
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: whoami
spec:
  entryPoints:
    - web
    - websecure
  routes:
    - match: Host(`example.com`) && PathPrefix(`/whoami`)
      kind: Rule
      services:
        - name: whoami
          port: 80
      middlewares:
        - name: traefik-real-ip
---
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: traefik-real-ip
spec:
  plugin:
    traefik-real-ip:
      forwardedForDepth: 2
---
```

For further details on using plugins with Traefik, consult the [official Traefik documentation](https://doc.traefik.io/traefik/plugins/).
