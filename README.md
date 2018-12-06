## Kubernetes Dynamic Admission Control Example based on Webhooks

This example is based on the [kubewebhook](https://github.com/slok/kubewebhook) Go framework from Slok. It checks pods (or anything that creates pods, e.g. a deployment/replicaset) for a key/value annotation pair and will admit or reject the pod creation accordingly. You can read more about dynamic admission control in Kubernetes [here](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks).

## Usage

Make sure your Kubernetes cluster meets the prerequisites as described [here](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#prerequisites). This example was tested against Kubernetes v1.12.

### Clone the Repository

```bash
$ git clone https://github.com/embano1/k8s-webhook-validator
$ cd k8s-webhook-validator
```

### Register the Validating Webhook

Before registering the webhook make sure to change the following sections in the YAML `deploy/register-hook.yaml` as per your environment and needs:

```yaml
webhooks:
  - name: [name]
    clientConfig:
      # replace with your webhook endpoint (can also be a service-exposed kubernetes deployment)
      url: [webhook_url]
      # replace with your base64 encoded CA cert for the webhook
      caBundle: [base64 encoded CA cert]
[...]
    failurePolicy: [Fail or Ignore]
```

```bash
$ kubectl create -f deploy/register-hook.yaml
validatingwebhookconfiguration.admissionregistration.k8s.io/pod-validate-webhook created
```

### Build the webhook server

This example uses Go modules so Go v1.11+ is required.

```bash
$ go build .
```

**Note:** If you want to run the webhook server inside Kubernetes you need to build a container image and deploy it accordingly. You also need to expose it as a `Service` and change `deploy/register-hook.yaml` as per guidelines [here](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#admission-webhooks).

### Run the webhook server as a standalone Binary

Note: Kubernetes requires secure communication to the webhook. So you need to generate a CA certificate and key. I recommend [mkcert](https://github.com/FiloSottile/mkcert) for local development and self-signed certificates.

```bash
# We will use the example deployments and match against an annotation key "signed" and value "true"
$ ./faas-validator -tls-cert-file <path_to_cert> -tls-key-file <path_to_cert_key> -key="signed" -value="true"
```

### Deploy the Example

```bash
$ kubectl create -f deploy/allowed_pod.yaml # this should succeed and a pod being created
$ kubectl create -f deploy/denied_pod.yaml # a deployment will be created but no pods
```

Observe the logs from the validator:

```bash
2018/12/06 17:30:32 [DEBUG] reviewing request 4033abaa-f974-11e8-8011-024247bc399e, named: default/
2018/12/06 17:30:32 [INFO] pod alpine-allowed-9dd5f8588-zxdvj is valid
2018/12/06 17:31:07 [DEBUG] reviewing request 54f4c52f-f974-11e8-8011-024247bc399e, named: default/
2018/12/06 17:31:07 [INFO] pod alpine-denied-685cb6cfb7-jk275 is not valid
```


