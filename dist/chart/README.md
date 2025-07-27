# OTTO Helm Chart

This Helm chart is used to deploy the OAuth Token Triage Operator (OTTO) in Kubernetes clusters. OTTO is a Kubernetes operator designed to manage and automate the lifecycle of OAuth tokens.

## Prerequisites

- A running Kubernetes cluster
- [Helm](https://helm.sh/) installed on your local machine
- `kubectl` configured to interact with your cluster

## Installation

To install the OTTO Helm chart, first add the Helm repository:

```bash
helm repo add otto https://winklermichael.github.io/otto
helm repo update
```

Then, install the chart:
```bash
helm install my-otto otto/otto
```

This will deploy OTTO into your Kubernetes cluster under the release name my-otto.

## Uninstallation
To uninstall the OTTO Helm chart, run:
```bash
helm uninstall my-otto
```

This will remove all resources associated with the OTTO release.

## Configuration
The chart supports various configuration options. You can customize the deployment by providing a values.yaml file or using --set flags during installation.

The operator can be configured using environment variables. The following variables are available:
- `REQUEUE_TIME`: The time after which the operator will requeue a resource for reconciliation in case of a retry. Default is `30s`.
- `HTTP_CLIENT_TIMEOUT`: The timeout for HTTP client requests. Default is `10s`.

### Example
```bash
helm install my-otto otto/otto --set key=value
```
For a full list of configurable options, refer to the values.yaml file in the chart.

## Usage
Once deployed, OTTO will manage OAuth tokens based on the OAuthTokenConfig custom resource. You can create and manage these resources to define token configurations.

## Development
To test the chart locally, you can use the following commands:

Lint the chart:
```bash
helm lint ./dist/chart
```

Install the chart locally:
```bash
helm install my-otto ./dist/chart --namespace otto-system --create-namespace
```

Check the release status:
```bash
helm status my-otto --namespace otto-system
```

## License
This Helm chart is licensed under the MIT License. See the LICENSE file for details.