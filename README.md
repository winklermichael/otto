# OAuth Token Triage Operator (OTTO)

The OAuth Token Triage Operator (OTTO) is a Kubernetes operator designed to manage and automate the lifecycle of OAuth tokens. It ensures that tokens are securely fetched, refreshed, and stored in Kubernetes secrets, enabling seamless integration with OAuth-based authentication systems. OTTO simplifies token management by handling credential-based token acquisition and refresh token workflows, ensuring tokens remain valid and up-to-date.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

### Prerequisites

This project requires that a kubernetes cluster is running and can be connected to.
Additionally kubectl needs to be configured to interact with the cluster.
[Helm.sh](https://helm.sh) needs to be installed on your machine.

In order to deploy this project, the helm repository needs to be added to your local installation:

```
helm repo add otto https://winklermichael.github.io/otto
helm repo update
```

If deployment should be handled via kustomize the needed dependencies need to be installed via:

```
make controller-gen
make kustomize
make setup-envtest
make golangci-lint
```

### Installing

Make sure your machine is set up properly by following the steps mentioned in the Prerequisites chapter.

To install the operator in your cluster run the following command.
```
helm install my-otto otto/otto
```

To uninstall the operator the operator from your cluster run the following command:

```
helm uninstall my-otto
```

Alternatively the operator can be deployed via kustomize using the provided make recipes.

First, build the project:

```
make build
```

Then build the docker image (and make sure it is accessible from your kubernetes cluster):

```
make docker-build
```

To deploy the CRDs to the cluster run the following make recipe:

```
make install
```

The CRDs can be uninstalled using:

```
make uninstall
```

Finally the operator can be deployed to the cluster with the following command:

```
make deploy
```

To undeploy the operator from the cluster run the following command:

```
make undeploy
```

## Running the tests

To run the tests a make recipe is provided:
```
make test
```

For end-to-end tests, run the following recipe:

```
make test-e2e
```

Additionally envtests are provided:

```
make envtest
```

## Deployment

For productive systems it is recommended to use the provided helm-chart. See Chapter Installing for more information.

## Configuration

The operator can be configured using environment variables. The following variables are available:
- `REQUEUE_TIME`: The time after which the operator will requeue a resource for reconciliation in case of a retry. Default is `30s`.
- `HTTP_CLIENT_TIMEOUT`: The timeout for HTTP client requests. Default is `10s`.

## Documentation
For detailed documentation on the OAuth Token Triage Operator, including API specifications, design decisions, and usage examples, please refer to the [docs](docs/) directory.

## Usage
For an example of how to use the OAuth Token Triage Operator, please refer to the [example](example/) directory. This directory contains sample configurations and usage patterns for integrating OTTO with your applications.

## Built With

* [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) - Framework for building Kubernetes APIs using CRDs

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/winklermichael/otto). 

## Authors

* **Michael Winkler** - *Initial work* - [winklermichael](https://github.com/winklermichael)

See also the list of [contributors](https://github.com/winklermichael/otto/contributors) who participated in this project.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

## Acknowledgments

* Thank you to Philipp Raith for supervising this project in its inception as a student project at the Technical University of Vienna

