package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	authv1alpha1 "github.com/winklermichael/otto/api/v1alpha1"
	definitions "github.com/winklermichael/otto/internal/controller/definitions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OAuthTokenConfigReconciler reconciles a OAuthTokenConfig object
type OAuthTokenConfigReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
	HTTPClient    *http.Client
}

var (
	REQUEUE_TIME        = getEnvDuration("REQUEUE_TIME", 30*time.Second)
	HTTP_CLIENT_TIMEOUT = getEnvDuration("HTTP_CLIENT_TIMEOUT", 10*time.Second)
)

// +kubebuilder:rbac:groups=auth.example.com,resources=oauthtokenconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.example.com,resources=oauthtokenconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.example.com,resources=oauthtokenconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update;delete

/* MAIN RECONCILER FUNCTION */

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OAuthTokenConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Create a logger for the current context
	log := log.FromContext(ctx)

	// Fetch the OAuthTokenConfig resource
	var oauthTokenConfig authv1alpha1.OAuthTokenConfig
	if err := r.fetchResource(ctx, req.NamespacedName, &oauthTokenConfig); err != nil {
		log.Error(err, "Failed to fetch OAuthTokenConfig", "OAuthTokenConfig", req.NamespacedName, "Error", err)
		r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceFetchFailed", fmt.Sprintf("Failed to fetch OAuthTokenConfig: %v", err))

		return ctrl.Result{}, err
	}

	// Emit an event indicating the reconciliation has started
	log.Info("Starting reconciliation")
	r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeNormal, "ReconciliationStarted", "Starting reconciliation")

	// Check if the current time is after the NextRefresh timestamp
	currentTime := time.Now()
	if !oauthTokenConfig.Status.NextRefresh.IsZero() && currentTime.Before(oauthTokenConfig.Status.NextRefresh.Time) {
		log.Info("Skipping reconciliation", "nextRefresh", oauthTokenConfig.Status.NextRefresh.Time)
		r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeNormal, "ReconciliationSkipped", fmt.Sprintf("Skipping reconciliation, next refresh: %s", oauthTokenConfig.Status.NextRefresh.Time))

		if time.Until(oauthTokenConfig.Status.NextRefresh.Time) <= 0 {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{
			RequeueAfter: time.Until(oauthTokenConfig.Status.NextRefresh.Time),
		}, nil
	}

	// Fetch the target secret
	targetSecretName := types.NamespacedName{
		Name:      oauthTokenConfig.Spec.Target.SecretRef.Name,
		Namespace: oauthTokenConfig.Spec.Target.SecretRef.Namespace,
	}
	targetSecret := &corev1.Secret{}
	if err := r.fetchResource(ctx, targetSecretName, targetSecret); client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to fetch TargetSecret", "TargetSecret", targetSecretName, "Error", err)
		r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceFetchFailed", fmt.Sprintf("Failed to fetch TargetSecret: %v", err))

		// Set CRD status to FAILED
		oauthTokenConfig.Status.Status = definitions.STATUS_FAILED
		if updateErr := r.updateStatus(ctx, &oauthTokenConfig); updateErr != nil {
			log.Error(updateErr, "Failed to update OAuthTokenConfig status", "Error", updateErr)
			r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceUpdateFailed", fmt.Sprintf("Failed to update OAuthTokenConfig status: %v", updateErr))
			return ctrl.Result{}, updateErr
		}

		return ctrl.Result{}, err
	}

	// Fetch the credentials secret
	credentialsSecretName := types.NamespacedName{
		Name:      oauthTokenConfig.Spec.Credentials.SecretRef.Name,
		Namespace: oauthTokenConfig.Spec.Credentials.SecretRef.Namespace,
	}
	credentialsSecret := &corev1.Secret{}
	if err := r.fetchResource(ctx, credentialsSecretName, credentialsSecret); err != nil {
		log.Error(err, "Failed to fetch CredentialsSecret", "CredentialsSecret", credentialsSecretName, "Error", err)
		r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceFetchFailed", fmt.Sprintf("Failed to fetch CredentialsSecret: %v", err))

		// Set CRD status to FAILED
		oauthTokenConfig.Status.Status = definitions.STATUS_FAILED
		if updateErr := r.updateStatus(ctx, &oauthTokenConfig); updateErr != nil {
			log.Error(updateErr, "Failed to update OAuthTokenConfig status", "Error", updateErr)
			r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceUpdateFailed", fmt.Sprintf("Failed to update OAuthTokenConfig status: %v", updateErr))
			return ctrl.Result{}, updateErr
		}

		return ctrl.Result{}, err
	}
	// Validate the credentials secret
	if err := r.validateCredentialsSecret(ctx, oauthTokenConfig, *credentialsSecret); err != nil {
		log.Error(err, "Credentials secret validation failed", "CredentialsSecret", credentialsSecretName, "Error", err)
		r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceValidationFailed", fmt.Sprintf("Credentials secret validation failed: %v", err))

		// Set CRD status to FAILED
		oauthTokenConfig.Status.Status = definitions.STATUS_FAILED
		if updateErr := r.updateStatus(ctx, &oauthTokenConfig); updateErr != nil {
			log.Error(updateErr, "Failed to update OAuthTokenConfig status", "Error", updateErr)
			r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceUpdateFailed", fmt.Sprintf("Failed to update OAuthTokenConfig status: %v", updateErr))
			return ctrl.Result{}, updateErr
		}

		// Requeue the reconciliation after a short delay to retry
		return ctrl.Result{RequeueAfter: REQUEUE_TIME}, err
	}

	// Get current timestamp
	now := metav1.Now()

	// Fetch new tokens
	tokens, err := r.refreshToken(ctx, oauthTokenConfig, *targetSecret, *credentialsSecret)
	if err != nil {
		log.Error(err, "Failed to refresh token", "Error", err)
		r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "TokenRefreshFailed", fmt.Sprintf("Failed to refresh token: %v", err))

		// Set CRD status to FAILED
		oauthTokenConfig.Status.Status = definitions.STATUS_FAILED
		if updateErr := r.updateStatus(ctx, &oauthTokenConfig); updateErr != nil {
			log.Error(updateErr, "Failed to update OAuthTokenConfig status", "Error", updateErr)
			r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceUpdateFailed", fmt.Sprintf("Failed to update OAuthTokenConfig status: %v", updateErr))
			return ctrl.Result{}, updateErr
		}

		// Requeue the reconciliation after a short delay to retry
		return ctrl.Result{RequeueAfter: REQUEUE_TIME}, err
	}
	log.Info("Tokens refreshed successfully")

	// Update/Create target secret
	targetSecretExists := targetSecret.Data != nil
	if !targetSecretExists {
		targetSecret.Data = make(map[string][]byte)
		targetSecret.Name = oauthTokenConfig.Spec.Target.SecretRef.Name
		targetSecret.Namespace = oauthTokenConfig.Spec.Target.SecretRef.Namespace
	}
	targetSecret.Data[oauthTokenConfig.Spec.Target.AccessTokenFieldName] = []byte(tokens.AccessToken)
	targetSecret.Data[oauthTokenConfig.Spec.Target.RefreshTokenFieldName] = []byte(tokens.RefreshToken)

	if !targetSecretExists {
		if err := r.createResource(ctx, targetSecret); err != nil {
			log.Error(err, "Failed to create target secret", "TargetSecret", targetSecret.Name, "Error", err)
			r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceCreationFailed", fmt.Sprintf("Failed to create target secret: %v", err))

			// Set CRD status to FAILED
			oauthTokenConfig.Status.Status = definitions.STATUS_FAILED
			if updateErr := r.updateStatus(ctx, &oauthTokenConfig); updateErr != nil {
				log.Error(updateErr, "Failed to update OAuthTokenConfig status", "Error", updateErr)
				r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceUpdateFailed", fmt.Sprintf("Failed to update OAuthTokenConfig status: %v", updateErr))
				return ctrl.Result{}, updateErr
			}

			return ctrl.Result{}, err
		}
		r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeNormal, "ResourceCreated", fmt.Sprintf("Target secret %s created successfully", targetSecret.Name))
	} else {
		if err := r.updateResource(ctx, targetSecret); err != nil {
			log.Error(err, "Failed to update target secret", "TargetSecret", targetSecret.Name, "Error", err)
			r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceUpdateFailed", fmt.Sprintf("Failed to update target secret: %v", err))

			// Set CRD status to FAILED
			oauthTokenConfig.Status.Status = definitions.STATUS_FAILED
			if updateErr := r.updateStatus(ctx, &oauthTokenConfig); updateErr != nil {
				log.Error(updateErr, "Failed to update OAuthTokenConfig status", "Error", updateErr)
				r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceUpdateFailed", fmt.Sprintf("Failed to update OAuthTokenConfig status: %v", updateErr))
				return ctrl.Result{}, updateErr
			}

			return ctrl.Result{}, err
		}
		r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeNormal, "ResourceUpdated", fmt.Sprintf("Target secret %s updated successfully", targetSecret.Name))
	}

	// Update CRD
	oauthTokenConfig.Status.LastRefresh = now
	oauthTokenConfig.Status.ExpirationTime = metav1.NewTime(now.Add(time.Duration(tokens.ExpiresIn) * time.Second))
	oauthTokenConfig.Status.NextRefresh = metav1.NewTime(oauthTokenConfig.Status.ExpirationTime.Time.Add(-(time.Duration(float64(tokens.ExpiresIn) * float64(time.Second) * (float64(oauthTokenConfig.Spec.RefreshBufferPercentage) / 100)))))
	oauthTokenConfig.Status.RefreshExpirationTime = metav1.NewTime(now.Add(time.Duration(tokens.RefreshExpiresIn) * time.Second))
	oauthTokenConfig.Status.Status = definitions.STATUS_REFRESHED

	if err := r.updateStatus(ctx, &oauthTokenConfig); err != nil {
		log.Error(err, "Failed to update OAuthTokenConfig", "Error", err)
		r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceUpdateFailed", fmt.Sprintf("Failed to update OAuthTokenConfig: %v", err))

		// Set CRD status to FAILED
		oauthTokenConfig.Status.Status = definitions.STATUS_FAILED
		if updateErr := r.updateStatus(ctx, &oauthTokenConfig); updateErr != nil {
			log.Error(updateErr, "Failed to update OAuthTokenConfig status", "Error", updateErr)
			r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeWarning, "ResourceUpdateFailed", fmt.Sprintf("Failed to update OAuthTokenConfig status: %v", updateErr))
			return ctrl.Result{}, updateErr
		}

		return ctrl.Result{}, err
	}

	// Finalize Reconciliation
	log.Info("Reconciliation completed successfully")
	r.EventRecorder.Event(&oauthTokenConfig, corev1.EventTypeNormal, "ReconciliationSuccessful", "Reconciliation completed successfully")

	// Refresh the controller after the specified refresh interval if set, else calculate based on token expiration
	requeueAfter := time.Duration(0)
	if oauthTokenConfig.Spec.RefreshInterval != nil && oauthTokenConfig.Spec.RefreshInterval.Duration > 0 {
		requeueAfter = oauthTokenConfig.Spec.RefreshInterval.Duration
	} else {
		requeueAfter = time.Until(oauthTokenConfig.Status.NextRefresh.Time)
	}

	return ctrl.Result{
		RequeueAfter: requeueAfter, // Requeue after the specified refresh interval
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OAuthTokenConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecorder = mgr.GetEventRecorderFor("OAuthTokenConfigController")

	// Initialize HTTPClient if it is nil
	if r.HTTPClient == nil {
		r.HTTPClient = &http.Client{
			Timeout: HTTP_CLIENT_TIMEOUT, // Set a reasonable timeout
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.OAuthTokenConfig{}).
		Complete(r)
}
