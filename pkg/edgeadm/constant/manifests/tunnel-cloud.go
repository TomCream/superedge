/*
Copyright 2020 The SuperEdge Authors.

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

package manifests

const APP_TUNNEL_CLOUD = "tunnel-cloud.yaml"

const TunnelCloudYaml = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: tunnel-cloud
  namespace: kube-system
rules:
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "update"]
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["get"]
  - apiGroups: [""]
    resources: ["services"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: tunnel-cloud
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tunnel-cloud
subjects:
  - kind: ServiceAccount
    name: tunnel-cloud
    namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tunnel-cloud
  namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: tunnel-cloud-conf
  namespace: kube-system
data:
  mode.toml: |
    [mode]
        [mode.cloud]
            [mode.cloud.stream]
                [mode.cloud.stream.server]
                    grpcport = 9000
                    logport = 51010
                    key = "/etc/superedge/tunnel/certs/tunnel-cloud-server.key"
                    cert = "/etc/superedge/tunnel/certs/tunnel-cloud-server.crt"
                    tokenfile = "/etc/superedge/tunnel/token/token"
                [mode.cloud.stream.dns]
                     configmap="tunnel-nodes"
                     hosts = "/etc/superedge/tunnel/nodes/hosts"
                     service = "tunnel-cloud"
            [mode.cloud.tcp]
                "0.0.0.0:6443" = "127.0.0.1:6443"
            [mode.cloud.https]
                cert ="/etc/superedge/tunnel/certs/apiserver-kubelet-server.crt"
                key = "/etc/superedge/tunnel/certs/apiserver-kubelet-server.key"
                [mode.cloud.https.addr]
                    "10250" = "127.0.0.1:10250"
                    "10300" = "127.0.0.1:10250"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: tunnel-cloud-token
  namespace: kube-system
data:
  token: |
    default:{{.TunnelCloudEdgeToken}}
---
apiVersion: v1
data:
  tunnel-cloud-server.crt: '{{.TunnelPersistentConnectionServerCrt}}'
  tunnel-cloud-server.key: '{{.TunnelPersistentConnectionServerKey}}'
  apiserver-kubelet-server.crt: '{{.TunnelProxyServerCrt}}'
  apiserver-kubelet-server.key: '{{.TunnelProxyServerKey}}'
kind: Secret
metadata:
  name: tunnel-cloud-cert
  namespace: kube-system
type: Opaque
---
apiVersion: v1
kind: Service
metadata:
  name: tunnel-cloud
  namespace: kube-system
spec:
  ports:
    - name: proxycloud
      port: 9000
      protocol: TCP
      targetPort: 9000
  selector:
    app: tunnel-cloud
  type: NodePort
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: tunnel-cloud
  name: tunnel-cloud
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: tunnel-cloud
  template:
    metadata:
      labels:
        app: tunnel-cloud
    spec:
      serviceAccount: tunnel-cloud
      serviceAccountName: tunnel-cloud
      containers:
        - name: tunnel-cloud
          image: superedge/tunnel:v0.1.0
          imagePullPolicy: IfNotPresent
          livenessProbe:
            httpGet:
              path: /cloud/healthz
              port: 51010
            initialDelaySeconds: 10
            periodSeconds: 60
            timeoutSeconds: 3
            successThreshold: 1
            failureThreshold: 1
          command:
            - /usr/local/bin/tunnel
          args:
            - --m=cloud
            - --c=/etc/superedge/tunnel/conf/mode.toml
            - --log-dir=/var/log/tunnel
            - --alsologtostderr
          env:
            - name: POD_IP
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: status.podIP
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: metadata.namespace
          volumeMounts:
            - name: token
              mountPath: /etc/superedge/tunnel/token
            - name: certs
              mountPath: /etc/superedge/tunnel/certs
            - name: hosts
              mountPath: /etc/superedge/tunnel/nodes
            - name: conf
              mountPath: /etc/superedge/tunnel/conf
          ports:
            - containerPort: 9000
              name: tunnel
              protocol: TCP
            - containerPort: 7000
              name: gateway
              protocol: TCP
            - containerPort: 10250
              name: kubelet
              protocol: TCP
            - containerPort: 6443
              name: apiserver
              protocol: TCP
          resources:
            limits:
              cpu: 50m
              memory: 100Mi
            requests:
              cpu: 10m
              memory: 20Mi
      volumes:
        - name: token
          configMap:
            name: tunnel-cloud-token
        - name: certs
          secret:
            secretName: tunnel-cloud-cert
        - name: hosts
          configMap:
            name: tunnel-nodes
        - name: conf
          configMap:
            name: tunnel-cloud-conf
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
        - key: "node-role.kubernetes.io/master"
          operator: "Exists"
          effect: "NoSchedule"
`