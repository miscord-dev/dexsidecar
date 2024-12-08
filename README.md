## Dex Sidecar for Kubernetes
## What this does?
* Issue access_token from dex using Kubernetes ServiceAccount token
    * [Dex Machine Authentication](https://dexidp.io/docs/guides/token-exchange/)
    * [ServiceAccount token volume projection](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#serviceaccount-token-volume-projection)
* Why we don't set Kubernetes directly to the provider of clients?
    * Some clients doesn't allow to specify multiple clients.
    * Combine multiple connectors with Dex

## Example
* Example Dex config
```yaml
...
connectors:
- type: oidc
  id: k3s
  name: k8s.tsuzu.dev
  config:
    issuer: https://k8s.tsuzu.dev:6443
    clientID: dex
    scopes:
      - openid
      - federated:id
    userNameKey: sub
    getUserInfo: false
    issuerAlias: https://kubernetes.default.svc.cluster.local
    insecureSkipVerify: true
oauth2:
  skipApprovalScreen: true
  grantTypes:
    - "authorization_code"
    - "urn:ietf:params:oauth:grant-type:token-exchange"
    - "urn:ietf:params:oauth:grant-type:device_code"
staticClients:
- id: incus
  redirectURIs:
    - 'https://incus.tsuzu.dev:8443/oidc/callback'
    - '/device/callback'
  name: 'Incus'
  public: true
```

* Example Kubernetes manifest
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 1
  template:
    metadata:
      labels:
        app: nginx
    spec:
      initContainers:
        - name: init-incus
          image: ghcr.io/miscord-dev/dexsidecar@sha256:40fd19cc52734740277a805f4a56db5684965275f8dd7c996d4f2496150018e0
          restartPolicy: Always
          env:
            - name: dex_access_token_file
              value: /var/run/secrets/miscord.win/dex/token
            - name: dex_endpoint
              value: "https://dex.tsuzu.dev/token"
            - name: dex_basic_auth
              value: "user:"
            - name: dex_connector_id
              value: k3s
            - name: dex_grant_type
              value: urn:ietf:params:oauth:grant-type:token-exchange
            - name: dex_scope
              value: "openid federated_id"
            - name: dex_requested_token_type
              value: urn:ietf:params:oauth:token-type:access_token
            - name: dex_file_subject_token
              value: /var/run/secrets/kubernetes.io/dex/token
            - name: dex_subject_token_type
              value: urn:ietf:params:oauth:token-type:id_token
          volumeMounts:
            - name: incus-api-key
              mountPath: /var/run/secrets/miscord.win/dex
            - name: dex
              mountPath: /var/run/secrets/kubernetes.io/dex
      containers:
        - name: manager
          image: nginx
          volumeMounts:
            - name: incus-api-key
              mountPath: /var/run/secrets/miscord.win/dex
      volumes:
        - name: incus-api-key
          emptyDir: {}
        - name: dex
          projected:
            defaultMode: 420
            sources:
            - serviceAccountToken:
                audience: dex
                expirationSeconds: 7200
                path: token

```
