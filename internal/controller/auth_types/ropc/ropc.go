package ropc

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
	definitions "github.com/winklermichael/otto/internal/controller/definitions"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Function to handle ROPC refresh
func HandleRefresh(ctx context.Context, client *http.Client, oauthTokenConfig authv1alpha1.OAuthTokenConfig, targetSecret corev1.Secret, credentialsSecret corev1.Secret) (*definitions.Tokens, error) {
	// Extract client ID and client secret from the credentials secret
	clientID := string(credentialsSecret.Data[oauthTokenConfig.Spec.Credentials.ClientIDFieldName])
	clientSecret := string(credentialsSecret.Data[oauthTokenConfig.Spec.Credentials.ClientSecretFieldName])
	tokenURL := oauthTokenConfig.Spec.TokenURL

	// If targetSecret is empty refresh using the client credentials else check if the refresh token in target secret is expired, if not use it, if it is expired use client credentials
	refreshToken := string(targetSecret.Data[oauthTokenConfig.Spec.Target.RefreshTokenFieldName])
	if refreshToken == "" || time.Now().After(oauthTokenConfig.Status.RefreshExpirationTime.Time) {
		// Extract username and password from the credentials secret
		username := string(credentialsSecret.Data[oauthTokenConfig.Spec.Credentials.UsernameFieldName])
		password := string(credentialsSecret.Data[oauthTokenConfig.Spec.Credentials.PasswordFieldName])

		// Use client credentials to get a new access token
		data := url.Values{}
		data.Set(oauthTokenConfig.Spec.TokenRequest.GrantTypeFieldName, "password")
		data.Set(oauthTokenConfig.Spec.TokenRequest.ClientIDFieldName, clientID)
		data.Set(oauthTokenConfig.Spec.TokenRequest.ClientSecretFieldName, clientSecret)
		data.Set(oauthTokenConfig.Spec.TokenRequest.UsernameFieldName, username)
		data.Set(oauthTokenConfig.Spec.TokenRequest.PasswordFieldName, password)

		return getToken(ctx, client, oauthTokenConfig, tokenURL, data)
	}

	// Use the refresh token to get a new access token
	data := url.Values{}
	data.Set(oauthTokenConfig.Spec.TokenRequest.GrantTypeFieldName, "refresh_token")
	data.Set(oauthTokenConfig.Spec.TokenRequest.ClientIDFieldName, clientID)
	data.Set(oauthTokenConfig.Spec.TokenRequest.ClientSecretFieldName, clientSecret)
	data.Set(oauthTokenConfig.Spec.TokenRequest.RefreshTokenFieldName, refreshToken)

	return getToken(ctx, client, oauthTokenConfig, tokenURL, data)
}

// Function to get token
func getToken(ctx context.Context, client *http.Client, oauthTokenConfig authv1alpha1.OAuthTokenConfig, tokenURL string, data url.Values) (*definitions.Tokens, error) {
	log := log.FromContext(ctx)
	log.V(1).Info("Getting token", "tokenURL", tokenURL, "grantType", data.Get(oauthTokenConfig.Spec.TokenRequest.GrantTypeFieldName))

	// Build Request
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		log.Error(err, "Failed to create HTTP request", "tokenURL", tokenURL)
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set Content-Type header
	req.Header.Set("Content-Type", oauthTokenConfig.Spec.TokenRequest.ContentType)

	// Set additional headers if provided
	if oauthTokenConfig.Spec.TokenRequest.Headers != nil {
		for key, value := range oauthTokenConfig.Spec.TokenRequest.Headers {
			req.Header.Set(key, value)
		}
	}

	// Send Request
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Failed to make HTTP request", "tokenURL", tokenURL)
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error(closeErr, "Failed to close response body")
		}
	}()

	// Process Response
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		log.Info("Non-200 response received", "statusCode", resp.StatusCode, "body", string(responseBody))
		return nil, fmt.Errorf("non-200 response: %d, body: %s", resp.StatusCode, string(responseBody))
	}

	// Parse token response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "Failed to read response body")
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return parseTokenResponse(oauthTokenConfig, responseBody)
}

// Function to parse token response
func parseTokenResponse(oauthTokenConfig authv1alpha1.OAuthTokenConfig, responseBody []byte) (*definitions.Tokens, error) {
	// Parse the response body into a generic map
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Create a Tokens struct to hold the parsed values
	tokens := &definitions.Tokens{}
	stringFieldMapping := map[string]*string{
		oauthTokenConfig.Spec.TokenResponse.AccessTokenFieldName:  &tokens.AccessToken,
		oauthTokenConfig.Spec.TokenResponse.RefreshTokenFieldName: &tokens.RefreshToken,
	}
	intFieldMapping := map[string]*int{
		oauthTokenConfig.Spec.TokenResponse.ExpirationFieldName:        &tokens.ExpiresIn,
		oauthTokenConfig.Spec.TokenResponse.RefreshExpirationFieldName: &tokens.RefreshExpiresIn,
	}

	// Map string fields
	for fieldName, target := range stringFieldMapping {
		if value, ok := response[fieldName]; ok {
			if strValue, ok := value.(string); ok {
				*target = strValue
			} else {
				return nil, fmt.Errorf("field '%s' is not a string", fieldName)
			}
		} else {
			return nil, fmt.Errorf("required field '%s' not found in response", fieldName)
		}
	}

	// Map integer fields
	for fieldName, target := range intFieldMapping {
		if value, ok := response[fieldName]; ok {
			if floatValue, ok := value.(float64); ok {
				*target = int(floatValue) // JSON numbers are float64 by default
			} else {
				return nil, fmt.Errorf("field '%s' is not a number", fieldName)
			}
		} else {
			return nil, fmt.Errorf("required field '%s' not found in response", fieldName)
		}
	}

	return tokens, nil
}
