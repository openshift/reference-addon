apiVersion: apps/v1
kind: Deployment
metadata:
  name: reference-addon
  namespace: reference-addon
  labels:
    app.kubernetes.io/name: reference-addon
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: reference-addon
  template:
    metadata:
      labels:
        app.kubernetes.io/name: reference-addon
    spec:
      serviceAccountName: reference-addon
      containers:
      - name: manager
        image: quay.io/openshift/reference-addon:latest
        args:
        - --enable-leader-election
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
