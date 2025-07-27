package controller

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	authv1alpha1 "github.com/winklermichael/otto/api/v1alpha1"
	ropc "github.com/winklermichael/otto/internal/controller/auth_types/ropc"
	definitions "github.com/winklermichael/otto/internal/controller/definitions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

/* HELPER FUNCTIONS */

// function to read duration from env or use default
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if parsedValue, err := strconv.Atoi(value); err == nil {
			return time.Duration(parsedValue) * time.Second
		}
	}
	return defaultValue
}

// function to fetch a resource by name
func (r *OAuthTokenConfigReconciler) fetchResource(ctx context.Context, name types.NamespacedName, obj client.Object) error {
	log := log.FromContext(ctx)
	log.V(1).Info("Fetching resource", "name", name, "type", fmt.Sprintf("%T", obj))
	if err := r.Get(ctx, name, obj); err != nil {
		log.V(1).Info("Failed to fetch resource", "name", name, "type", fmt.Sprintf("%T", obj), "error", err)
		return err
	}
	log.V(1).Info("Resource fetched successfully", "name", name, "type", fmt.Sprintf("%T", obj))
	return nil
}

// function to create a resource
func (r *OAuthTokenConfigReconciler) createResource(ctx context.Context, obj client.Object) error {
	log := log.FromContext(ctx)
	log.V(1).Info("Creating resource", "name", obj.GetName(), "type", fmt.Sprintf("%T", obj))
	if err := r.Create(ctx, obj); err != nil {
		log.V(1).Info("Failed to create resource", "name", obj.GetName(), "type", fmt.Sprintf("%T", obj), "error", err)
		return err
	}
	log.V(1).Info("Resource created successfully", "name", obj.GetName(), "type", fmt.Sprintf("%T", obj))
	return nil
}

// function to update a resource
func (r *OAuthTokenConfigReconciler) updateResource(ctx context.Context, obj client.Object) error {
	log := log.FromContext(ctx)
	log.V(1).Info("Updating resource", "name", obj.GetName(), "type", fmt.Sprintf("%T", obj))
	if err := r.Update(ctx, obj); err != nil {
		log.V(1).Info("Failed to update resource", "name", obj.GetName(), "type", fmt.Sprintf("%T", obj), "error", err)
		return err
	}
	log.V(1).Info("Resource updated successfully", "name", obj.GetName(), "type", fmt.Sprintf("%T", obj))

	return nil
}

// function to update the status of a resource
func (r *OAuthTokenConfigReconciler) updateStatus(ctx context.Context, obj client.Object) error {
	log := log.FromContext(ctx)
	log.V(1).Info("Updating resource status", "name", obj.GetName(), "type", fmt.Sprintf("%T", obj))
	if err := r.Status().Update(ctx, obj); err != nil {
		log.V(1).Info("Failed to update resource status", "name", obj.GetName(), "type", fmt.Sprintf("%T", obj), "error", err)
		return err
	}
	log.V(1).Info("Resource status updated successfully", "name", obj.GetName(), "type", fmt.Sprintf("%T", obj))
	return nil
}

// function to validate credentials secret
func (r *OAuthTokenConfigReconciler) validateCredentialsSecret(ctx context.Context, oauthTokenConfig authv1alpha1.OAuthTokenConfig, credentialsSecret corev1.Secret) error {
	log := log.FromContext(ctx)
	log.V(1).Info("Validating credentials secret", "name", credentialsSecret.Name, "namespace", credentialsSecret.Namespace)

	// Check if the credentials secret contains the required fields
	missingFields := []string{}

	if _, ok := credentialsSecret.Data[oauthTokenConfig.Spec.Credentials.ClientIDFieldName]; !ok {
		missingFields = append(missingFields, oauthTokenConfig.Spec.Credentials.ClientIDFieldName)
	}
	if _, ok := credentialsSecret.Data[oauthTokenConfig.Spec.Credentials.ClientSecretFieldName]; !ok {
		missingFields = append(missingFields, oauthTokenConfig.Spec.Credentials.ClientSecretFieldName)
	}
	if oauthTokenConfig.Spec.Type == "ropc" {
		if _, ok := credentialsSecret.Data[oauthTokenConfig.Spec.Credentials.UsernameFieldName]; !ok {
			missingFields = append(missingFields, oauthTokenConfig.Spec.Credentials.UsernameFieldName)
		}
		if _, ok := credentialsSecret.Data[oauthTokenConfig.Spec.Credentials.PasswordFieldName]; !ok {
			missingFields = append(missingFields, oauthTokenConfig.Spec.Credentials.PasswordFieldName)
		}
	}

	if len(missingFields) > 0 {
		// Log the error and return it
		log.V(1).Info("Missing required fields in credentials secret", "name", credentialsSecret.Name, "namespace", credentialsSecret.Namespace, "missingFields", missingFields)
		err := fmt.Errorf("credentials secret %s/%s is missing required fields: %v", credentialsSecret.Namespace, credentialsSecret.Name, strings.Join(missingFields, ", "))
		return err
	}

	log.V(1).Info("Credentials secret validated successfully", "name", credentialsSecret.Name, "namespace", credentialsSecret.Namespace)
	return nil
}

// function to refresh token
func (r *OAuthTokenConfigReconciler) refreshToken(ctx context.Context, oauthTokenConfig authv1alpha1.OAuthTokenConfig, targetSecret corev1.Secret, credentialsSecret corev1.Secret) (*definitions.Tokens, error) {

	// Decide which grant type to use based on the OAuthTokenConfig spec
	if oauthTokenConfig.Spec.Type == "ropc" {
		return ropc.HandleRefresh(ctx, r.HTTPClient, oauthTokenConfig, targetSecret, credentialsSecret)
	}
	// If the type is not recognized, return an error
	return nil, fmt.Errorf("Unsupported OAuth2 grant type: %s", oauthTokenConfig.Spec.Type)
}
