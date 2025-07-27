# Design Document for OTTO

## Overview

OTTO is a Kubernetes operator built using the Kubebuilder framework. It manages OAuth token configurations through a custom resource definition (CRD) called `OAuthTokenConfig`. This document outlines the architectural design, key components, and workflows of the project.

---

## Architecture

### Key Components

1. **Custom Resource Definition (CRD)**:
   - `OAuthTokenConfig`: Defines the schema for managing OAuth token configurations.
   - Located in [api/v1alpha1/oauthtokenconfig_types.go](../api/v1alpha1/oauthtokenconfig_types.go).

2. **Controller**:
   - Watches for changes to `OAuthTokenConfig` resources and reconciles them.
   - Implements the core logic for managing OAuth tokens.
   - Entry point: [internal/controller/oauthtokenconfig_controller.go](../internal/controller/oauthtokenconfig_controller.go).

3. **Helm Chart**:
   - Provides a Helm chart for deploying OTTO in Kubernetes clusters.
   - Located in [dist/chart/Chart.yaml](../dist/chart/Chart.yaml).

---

## Workflow

### Reconciliation Loop

1. **Watch Events**:
   - The controller watches for create, update, and delete events on `OAuthTokenConfig` resources.

2. **Reconcile Logic**:
   - Validates the resource.
   - Retrieves secrets referenced in the `OAuthTokenConfig`.
   - Generates or refreshes OAuth tokens.
   - Updates the target secret with the generated tokens.

3. **Error Handling**:
   - Logs errors and retries reconciliation based on exponential backoff.

---

## Configuration

### Kustomize

- The project uses Kustomize for managing Kubernetes manifests.
- Key configurations:
  - [config/default/kustomization.yaml](../config/default/kustomization.yaml): Default deployment configuration.
  - [config/manager/kustomization.yaml](../config/manager/kustomization.yaml): Manager-specific configuration.

### Helm

- The Helm chart is used for deploying OTTO in Kubernetes clusters.
- Key files:
  - [dist/chart/Chart.yaml](../dist/chart/Chart.yaml): Chart metadata.
  - [dist/chart/templates/deployment.yaml](../dist/chart/templates/deployment.yaml): Deployment template for the operator. 

### Makefile

- The `Makefile` provides targets for common tasks:
  - `make manifests`: Generate CRDs and RBAC manifests.
  - `make generate`: Generate Go code for CRDs.
  - `make build`: Build the manager binary.
  - `make test`: Run unit tests.
  - `make docker-build`: Build the Docker image for the operator.