package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	authv1alpha1 "github.com/winklermichael/otto/api/v1alpha1"
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
}

// +kubebuilder:rbac:groups=auth.example.com,resources=oauthtokenconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.example.com,resources=oauthtokenconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.example.com,resources=oauthtokenconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update;delete

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OAuthTokenConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the OAuthTokenConfig resource
	var oauthTokenConfig authv1alpha1.OAuthTokenConfig
	if err := r.Get(ctx, req.NamespacedName, &oauthTokenConfig); err != nil {
		log.Error(err, "Failed to get OAuthTokenConfig")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the current time is after the NextRefresh timestamp
	now := time.Now()
	if !oauthTokenConfig.Status.NextRefresh.IsZero() && now.Before(oauthTokenConfig.Status.NextRefresh.Time.Add(-10*time.Second)) {
		log.Info("Skipping reconciliation as the current time is before the next refresh", "now", now, "nextRefresh", oauthTokenConfig.Status.NextRefresh.Time)
		return ctrl.Result{}, nil
	}

	// Emit an event indicating the reconciliation has started
	r.emitEvent(&oauthTokenConfig, corev1.EventTypeNormal, "ReconciliationStarted", "Starting reconciliation")

	// Fetch the target secret to check for existing tokens
	targetSecret := &corev1.Secret{}
	targetSecretName := types.NamespacedName{
		Name:      oauthTokenConfig.Spec.TargetSecretRef.Name,
		Namespace: oauthTokenConfig.Spec.TargetSecretRef.Namespace,
	}
	err := r.Get(ctx, targetSecretName, targetSecret)
	if err != nil && client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get target secret", "targetSecretName", targetSecretName)
		r.emitEvent(&oauthTokenConfig, corev1.EventTypeWarning, "TargetSecretNotFound", fmt.Sprintf("Failed to fetch target secret: %s", targetSecretName))
		return ctrl.Result{}, err
	}

	var tokens *Tokens

	// Fetch the credentials secret
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := types.NamespacedName{
		Name:      oauthTokenConfig.Spec.CredentialsSecretRef.Name,
		Namespace: oauthTokenConfig.Spec.CredentialsSecretRef.Namespace,
	}
	if err := r.Get(ctx, credentialsSecretName, credentialsSecret); err != nil {
		log.Error(err, "Failed to get credentials secret", "credentialsSecretName", credentialsSecretName)
		r.emitEvent(&oauthTokenConfig, corev1.EventTypeWarning, "CredentialsSecretNotFound", fmt.Sprintf("Failed to fetch credentials secret: %s", credentialsSecretName))
		return ctrl.Result{}, err
	}

	// Extract client ID and client secret from the credentials secret
	clientID := string(credentialsSecret.Data[oauthTokenConfig.Spec.ClientIDFieldName])
	clientSecret := string(credentialsSecret.Data[oauthTokenConfig.Spec.ClientSecretFieldName])

	if err == nil {
		// Target secret exists, check for existing tokens
		refreshToken := string(targetSecret.Data[oauthTokenConfig.Spec.RefreshTokenFieldName])

		// Check refresh token expiration from the CRD status
		refreshExpiration := oauthTokenConfig.Status.RefreshExpiration.Time
		log.Info("Checking refresh token expiration", "refreshExpiration", refreshExpiration, "now", now)

		if now.Before(refreshExpiration) {
			log.Info("Refresh token is valid, using it to refresh access token")
			tokens, err = refreshAccessToken(clientID, clientSecret, refreshToken, oauthTokenConfig.Spec.RefreshURL)
			if err != nil {
				log.Error(err, "Failed to refresh token using refresh token, falling back to credentials")
				r.emitEvent(&oauthTokenConfig, corev1.EventTypeWarning, "TokenRefreshFailed", "Failed to refresh token using refresh token, falling back to credentials")
			} else {
				log.Info("Successfully refreshed access token using refresh token")
				r.emitEvent(&oauthTokenConfig, corev1.EventTypeNormal, "TokenRefreshSucceeded", "Successfully refreshed access token using refresh token")
			}
		} else {
			log.Info("Refresh token is expired, using credentials to fetch new tokens")
			tokens, err = r.fetchTokensUsingCredentials(ctx, &oauthTokenConfig, clientID, clientSecret)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Target secret does not exist, use username/password to get initial tokens
		log.Info("Target secret not found, using credentials to fetch initial tokens")
		tokens, err = r.fetchTokensUsingCredentials(ctx, &oauthTokenConfig, clientID, clientSecret)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Store the tokens in the target secret
	if err := r.storeTokensInSecret(ctx, &oauthTokenConfig, tokens); err != nil {
		return ctrl.Result{}, err
	}

	// Update the CRD status with token metadata
	if err := r.updateCRDStatus(ctx, &oauthTokenConfig, tokens); err != nil {
		return ctrl.Result{}, err
	}

	// Determine requeue interval
	requeueAfter := r.calculateRequeueInterval(&oauthTokenConfig, tokens)
	log.Info("Requeuing reconciliation", "requeueAfter", requeueAfter)
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// fetchTokensUsingCredentials fetches tokens using username and password
func (r *OAuthTokenConfigReconciler) fetchTokensUsingCredentials(ctx context.Context, oauthTokenConfig *authv1alpha1.OAuthTokenConfig, clientID, clientSecret string) (*Tokens, error) {
	log := log.FromContext(ctx)

	// Fetch the credentials secret
	credentialsSecret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Name:      oauthTokenConfig.Spec.CredentialsSecretRef.Name,
		Namespace: oauthTokenConfig.Spec.CredentialsSecretRef.Namespace,
	}
	if err := r.Get(ctx, secretName, credentialsSecret); err != nil {
		log.Error(err, "Failed to get credentials secret", "secretName", secretName)
		r.emitEvent(oauthTokenConfig, corev1.EventTypeWarning, "SecretNotFound", fmt.Sprintf("Failed to fetch credentials secret: %s", secretName))
		return nil, err
	}

	// Extract username and password from the credentials secret
	username := string(credentialsSecret.Data[oauthTokenConfig.Spec.UsernameFieldName])
	password := string(credentialsSecret.Data[oauthTokenConfig.Spec.PasswordFieldName])
	tokenURL := oauthTokenConfig.Spec.RefreshURL

	// Fetch tokens using username/password
	log.Info("Fetching tokens using credentials (username/password)")
	tokens, err := getInitialTokens(clientID, clientSecret, username, password, tokenURL)
	if err != nil {
		log.Error(err, "Failed to fetch tokens using credentials")
		r.emitEvent(oauthTokenConfig, corev1.EventTypeWarning, "TokenFetchFailed", "Failed to fetch tokens using credentials")
		oauthTokenConfig.Status.Status = "Failed"
		if updateErr := r.Status().Update(ctx, oauthTokenConfig); updateErr != nil {
			log.Error(updateErr, "Failed to update status")
		}
		return nil, fmt.Errorf("failed to fetch tokens using credentials: %w", err)
	}

	log.Info("Successfully fetched tokens using credentials")
	r.emitEvent(oauthTokenConfig, corev1.EventTypeNormal, "TokenRefreshSucceeded", "Successfully fetched tokens using credentials")
	return tokens, nil
}

// storeTokensInSecret stores tokens in the target secret
func (r *OAuthTokenConfigReconciler) storeTokensInSecret(ctx context.Context, oauthTokenConfig *authv1alpha1.OAuthTokenConfig, tokens *Tokens) error {
	if tokens == nil {
		return fmt.Errorf("tokens object is nil, cannot store tokens in secret")
	}

	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oauthTokenConfig.Spec.TargetSecretRef.Name,
			Namespace: oauthTokenConfig.Spec.TargetSecretRef.Namespace,
		},
		Data: map[string][]byte{
			oauthTokenConfig.Spec.TokenFieldName:        []byte(tokens.AccessToken),
			oauthTokenConfig.Spec.RefreshTokenFieldName: []byte(tokens.RefreshToken),
		},
	}
	return r.createOrUpdateSecret(ctx, targetSecret)
}

// updateCRDStatus updates the CRD status with token metadata
func (r *OAuthTokenConfigReconciler) updateCRDStatus(ctx context.Context, oauthTokenConfig *authv1alpha1.OAuthTokenConfig, tokens *Tokens) error {
	now := metav1.Now()
	expirationTime := metav1.NewTime(now.Add(time.Duration(tokens.ExpiresIn) * time.Second))
	buffer := time.Duration(float64(tokens.ExpiresIn) * float64(time.Second) * (float64(oauthTokenConfig.Spec.RefreshBufferPercentage) / 100))
	nextRefresh := metav1.NewTime(expirationTime.Time.Add(-buffer))
	refreshExpiration := metav1.NewTime(now.Add(time.Duration(tokens.RefreshExpiresIn) * time.Second))

	oauthTokenConfig.Status.LastRefreshed = now
	oauthTokenConfig.Status.ExpirationTime = expirationTime
	oauthTokenConfig.Status.NextRefresh = nextRefresh
	oauthTokenConfig.Status.RefreshExpiration = refreshExpiration
	oauthTokenConfig.Status.Status = "Refreshing"

	return r.Status().Update(ctx, oauthTokenConfig)
}

// calculateRequeueInterval calculates the requeue interval based on token expiration
func (r *OAuthTokenConfigReconciler) calculateRequeueInterval(oauthTokenConfig *authv1alpha1.OAuthTokenConfig, tokens *Tokens) time.Duration {
	if oauthTokenConfig.Spec.RefreshInterval != nil {
		return oauthTokenConfig.Spec.RefreshInterval.Duration
	}
	buffer := time.Duration(float64(tokens.ExpiresIn) * float64(time.Second) * (float64(oauthTokenConfig.Spec.RefreshBufferPercentage) / 100))
	return time.Duration(tokens.ExpiresIn)*time.Second - buffer
}

// emitEvent is a helper function to emit events
func (r *OAuthTokenConfigReconciler) emitEvent(obj runtime.Object, eventType, reason, message string) {
	r.EventRecorder.Event(obj, eventType, reason, message)
}

// createOrUpdateSecret creates or updates a Kubernetes Secret
func (r *OAuthTokenConfigReconciler) createOrUpdateSecret(ctx context.Context, secret *corev1.Secret) error {
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, existingSecret)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Secret does not exist, create it
			return r.Create(ctx, secret)
		}
		return err
	}

	// Secret exists, update it
	existingSecret.Data = secret.Data
	return r.Update(ctx, existingSecret)
}

// getInitialTokens fetches the initial access and refresh tokens using ROPC
func getInitialTokens(clientID, clientSecret, username, password, tokenURL string) (*Tokens, error) {
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("username", username)
	data.Set("password", password)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-200 response: %d, body: %s", resp.StatusCode, string(responseBody))
	}

	var tokens Tokens
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokens, nil
}

// refreshAccessToken refreshes the access token using the refresh token
func refreshAccessToken(clientID, clientSecret, refreshToken, tokenURL string) (*Tokens, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-200 response: %d, body: %s", resp.StatusCode, string(responseBody))
	}

	var tokens Tokens
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokens, nil
}

// Tokens represents the structure of the OAuth2 token response
type Tokens struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}

// SetupWithManager sets up the controller with the Manager.
func (r *OAuthTokenConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecorder = mgr.GetEventRecorderFor("OAuthTokenConfigController")
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.OAuthTokenConfig{}).
		Complete(r)
}
