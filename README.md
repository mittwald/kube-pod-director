# Kubernetes Pod director -- Kubernetes reverse proxy for stateful applications

This is a small reverse proxy designed to channel HTTP traffic to _one_ Pod matched by a Kubernetes service.

This project was built to enable a PHP application to run on Kubernetes, that had one sub-route
(an admin interface) that was not completely stateless.

The Pod director allows you to channel all traffic received by a service to _one_ Pod matched by a service
(usually, this is really bad design, but keep in mind that this project was created to work around architectural
failures in other projects).

## Usage

Consider you have a `Deployment`, a `Service` and an `Ingress` for a (largely) stateless application:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  selector:
    matchLabels:
      app: example
  replicas: 4
  template:
    metadata:
      labels:
        app: example
    spec:
      containers:
        - image: nginx
          name: web
          ports:
            - containerPort: 80
              name: http
---
apiVersion: v1
kind: Service
metadata:
  name: test
  namespace: default
spec:
  selector:
    app: example
  ports:
    - targetPort: 80
      port: 80
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test
spec:
  rules:
  - host: test.example
    http:
      paths:
      - backend:
          serviceName: test
          servicePort: 80
        path: /
```

Create a `Deployment` object with one replica of the `quay.io/spaces/pod-director` image;
pass it the previous service's name using the `-service` argument:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-director
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example-director
  replicas: 1
  template:
    metadata:
      labels:
        app: example-director
    spec:
      containers:
        - image: quay.io/spaces/pod-director
          name: director
          args:
            - "-namespace=default"
            - "-service=test"
            - "-logtostderr"
          ports:
            - containerPort: 80
              name: http
---
apiVersion: v1
kind: Service
metadata:
  name: test-director
  namespace: default
spec:
  selector:
    app: example-director
  ports:
    - targetPort: 8080
      port: 80
```

Then, configure your `Ingress` resource to forward certain routes to your director:

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test
spec:
  rules:
  - host: test.example
    http:
      paths:
      - path: /backend
        backend:
          serviceName: test-director
          servicePort: 80
      - path: /
        backend:
          serviceName: test
          servicePort: 80
```