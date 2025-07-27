package controller

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	authv1alpha1 "github.com/winklermichael/otto/api/v1alpha1"
	definitions "github.com/winklermichael/otto/internal/controller/definitions"
)

var _ = Describe("OAuthTokenConfig Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-crd"
			namespace         = "default"
			credentialsSecret = "test-credentials-secret"
			targetSecret      = "test-target-secret"
			clientIDField     = "client_id"
			clientSecretField = "client_secret"
			usernameField     = "username"
			passwordField     = "password"
			refreshTokenField = "refresh_token"
			accessTokenField  = "access_token"
			tokenURL          = "https://example.com/oauth/token"
		)

		ctx := context.Background()
		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}

		var mockServer *httptest.Server
		var receivedRequestBodies []map[string]interface{}

		BeforeEach(func() {
			// Initialize the slice to store received request bodies
			receivedRequestBodies = make([]map[string]interface{}, 0)

			// Create a mock HTTP server
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("POST"))
				Expect(r.URL.Path).To(Equal("/oauth/token"))

				// Read the request body
				bodyBytes, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())

				// Parse the request body as form data
				formData, err := url.ParseQuery(string(bodyBytes))
				Expect(err).NotTo(HaveOccurred())

				// Convert form data to a map
				bodyMap := make(map[string]interface{})
				for key, values := range formData {
					if len(values) > 0 {
						bodyMap[key] = values[0] // Use the first value for each key
					}
				}

				// Append the parsed body to the slice
				receivedRequestBodies = append(receivedRequestBodies, bodyMap)

				// Simulate a successful token response
				w.WriteHeader(http.StatusOK)
				_, err = w.Write([]byte(`{
					"access_token": "mock-access-token",
					"refresh_token": "mock-refresh-token",
					"expires_in": 360,
					"refresh_expires_in": 3600
				}`))
				Expect(err).NotTo(HaveOccurred()) // Check the error from w.Write
			}))

			// Create the credentials secret
			credentials := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      credentialsSecret,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					clientIDField:     []byte("test-client-id"),
					clientSecretField: []byte("test-client-secret"),
					usernameField:     []byte("test-username"),
					passwordField:     []byte("test-password"),
				},
			}
			Expect(k8sClient.Create(ctx, credentials)).To(Succeed())

			// Create the OAuthTokenConfig resource
			oauthTokenConfig := &authv1alpha1.OAuthTokenConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: authv1alpha1.OAuthTokenConfigSpec{
					TokenURL: mockServer.URL + "/oauth/token", // Use the mock server URL
					Type:     "ropc",
					Credentials: authv1alpha1.CredentialsConfig{
						SecretRef:             corev1.SecretReference{Name: credentialsSecret, Namespace: namespace},
						ClientIDFieldName:     clientIDField,
						ClientSecretFieldName: clientSecretField,
						UsernameFieldName:     usernameField,
						PasswordFieldName:     passwordField,
					},
					Target: authv1alpha1.TargetConfig{
						SecretRef:             corev1.SecretReference{Name: targetSecret, Namespace: namespace},
						AccessTokenFieldName:  accessTokenField,
						RefreshTokenFieldName: refreshTokenField,
					},
					RefreshBufferPercentage: 10,
				},
			}
			Expect(k8sClient.Create(ctx, oauthTokenConfig)).To(Succeed())
		})

		AfterEach(func() {
			// Stop the mock server
			mockServer.Close()

			// Cleanup the OAuthTokenConfig resource
			oauthTokenConfig := &authv1alpha1.OAuthTokenConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, oauthTokenConfig)
			if err == nil {
				Expect(k8sClient.Delete(ctx, oauthTokenConfig)).To(Succeed())
			}

			// Cleanup the credentials secret
			credentials := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: credentialsSecret, Namespace: namespace}, credentials)
			if err == nil {
				Expect(k8sClient.Delete(ctx, credentials)).To(Succeed())
			}

			// Cleanup the target secret
			target := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: targetSecret, Namespace: namespace}, target)
			if err == nil {
				Expect(k8sClient.Delete(ctx, target)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource with credentials the first time", func() {
			By("Reconciling the created resource")
			controllerReconciler := &OAuthTokenConfigReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				EventRecorder: record.NewFakeRecorder(10), // Initialize the EventRecorder
				HTTPClient:    mockServer.Client(),        // Use the mock HTTP client
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify that the target secret was created
			target := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: targetSecret, Namespace: namespace}, target)
			Expect(err).NotTo(HaveOccurred())
			Expect(target.Data).To(HaveKey(accessTokenField))
			Expect(target.Data).To(HaveKey(refreshTokenField))

			// Verify that the status of the OAuthTokenConfig resource was updated
			oauthTokenConfig := &authv1alpha1.OAuthTokenConfig{}
			err = k8sClient.Get(ctx, typeNamespacedName, oauthTokenConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(oauthTokenConfig.Status.Status).To(Equal(definitions.STATUS_REFRESHED))
			Expect(oauthTokenConfig.Status.LastRefresh.Time).To(BeTemporally("~", time.Now(), time.Minute))

			// Check if username and password were part of the data sent to mock server
			Expect(receivedRequestBodies).To(HaveLen(1))
			Expect(receivedRequestBodies[0][oauthTokenConfig.Spec.TokenRequest.UsernameFieldName]).To(Equal("test-username"))
			Expect(receivedRequestBodies[0][oauthTokenConfig.Spec.TokenRequest.PasswordFieldName]).To(Equal("test-password"))
		})

		It("should successfully reconcile the resource with refresh token the second time", func() {
			By("Reconciling the created resource again")
			controllerReconciler := &OAuthTokenConfigReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				EventRecorder: record.NewFakeRecorder(10), // Initialize the EventRecorder
				HTTPClient:    mockServer.Client(),        // Use the mock HTTP client
			}

			// First reconciliation to create the target secret
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			oauthTokenConfig := &authv1alpha1.OAuthTokenConfig{}
			err = k8sClient.Get(ctx, typeNamespacedName, oauthTokenConfig)
			Expect(err).NotTo(HaveOccurred())

			// Set next refresh time to now
			oauthTokenConfig.Status.NextRefresh = metav1.NewTime(time.Now())
			Expect(k8sClient.Status().Update(ctx, oauthTokenConfig)).To(Succeed())

			// Second reconciliation to refresh the token
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify that the target secret was updated with new tokens
			target := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: targetSecret, Namespace: namespace}, target)
			Expect(err).NotTo(HaveOccurred())
			Expect(target.Data).To(HaveKey(accessTokenField))
			Expect(target.Data).To(HaveKey(refreshTokenField))

			// Verify that the status of the OAuthTokenConfig resource was updated again
			Expect(err).NotTo(HaveOccurred())
			Expect(oauthTokenConfig.Status.Status).To(Equal(definitions.STATUS_REFRESHED))
			Expect(oauthTokenConfig.Status.LastRefresh.Time).To(BeTemporally("~", time.Now(), time.Minute))

			// Check if username and password were part of the data sent to mock server for the first request
			// and the second request should only contain the refresh token
			Expect(receivedRequestBodies).To(HaveLen(2))
			Expect(receivedRequestBodies[0][oauthTokenConfig.Spec.TokenRequest.UsernameFieldName]).To(Equal("test-username"))
			Expect(receivedRequestBodies[0][oauthTokenConfig.Spec.TokenRequest.PasswordFieldName]).To(Equal("test-password"))
			Expect(receivedRequestBodies[1][oauthTokenConfig.Spec.TokenRequest.RefreshTokenFieldName]).To(Equal("mock-refresh-token"))

		})

		It("should skip reconciliation if next refresh time is not reached", func() {
			By("Setting a next refresh time in the future")
			controllerReconciler := &OAuthTokenConfigReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				EventRecorder: record.NewFakeRecorder(10), // Initialize the EventRecorder
				HTTPClient:    mockServer.Client(),        // Use the mock HTTP client
			}

			oauthTokenConfig := &authv1alpha1.OAuthTokenConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, oauthTokenConfig)
			Expect(err).NotTo(HaveOccurred())

			// Set next refresh time to 1 hour in the future
			oauthTokenConfig.Status.NextRefresh = metav1.NewTime(time.Now().Add(1 * time.Hour))
			Expect(k8sClient.Status().Update(ctx, oauthTokenConfig)).To(Succeed())

			// Reconcile again
			result, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeNumerically("~", 1*time.Hour, 10*time.Second))

			// Verify that no new requests were sent to the mock server
			Expect(receivedRequestBodies).To(BeEmpty())
		})

		It("should reconcile with credentials again the second time if refresh token is expired", func() {
			By("Reconciling the created resource with expired refresh token")
			controllerReconciler := &OAuthTokenConfigReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				EventRecorder: record.NewFakeRecorder(10), // Initialize the EventRecorder
				HTTPClient:    mockServer.Client(),        // Use the mock HTTP client
			}

			// First reconciliation to create the target secret
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			oauthTokenConfig := &authv1alpha1.OAuthTokenConfig{}
			err = k8sClient.Get(ctx, typeNamespacedName, oauthTokenConfig)
			Expect(err).NotTo(HaveOccurred())

			// Set next refresh time to now
			oauthTokenConfig.Status.NextRefresh = metav1.NewTime(time.Now())
			Expect(k8sClient.Status().Update(ctx, oauthTokenConfig)).To(Succeed())

			// Simulate an expired refresh token by setting refresh expiration time to now
			oauthTokenConfig.Status.RefreshExpirationTime = metav1.NewTime(time.Now())
			Expect(k8sClient.Status().Update(ctx, oauthTokenConfig)).To(Succeed())

			// Second reconciliation to refresh the token
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify that the status of the OAuthTokenConfig resource was updated again
			Expect(err).NotTo(HaveOccurred())
			Expect(oauthTokenConfig.Status.Status).To(Equal(definitions.STATUS_REFRESHED))
			Expect(oauthTokenConfig.Status.LastRefresh.Time).To(BeTemporally("~", time.Now(), time.Minute))

			// Check if username and password were part of the data sent to mock server for the first request
			// and the second request should only contain the refresh token
			Expect(receivedRequestBodies).To(HaveLen(2))
			Expect(receivedRequestBodies[0][oauthTokenConfig.Spec.TokenRequest.UsernameFieldName]).To(Equal("test-username"))
			Expect(receivedRequestBodies[0][oauthTokenConfig.Spec.TokenRequest.PasswordFieldName]).To(Equal("test-password"))
			Expect(receivedRequestBodies[1][oauthTokenConfig.Spec.TokenRequest.UsernameFieldName]).To(Equal("test-username"))
			Expect(receivedRequestBodies[1][oauthTokenConfig.Spec.TokenRequest.PasswordFieldName]).To(Equal("test-password"))
		})

		It("should emit event if token refresh failed", func() {
			By("Simulating a token refresh failure")
			// Create a mock HTTP server that returns an error
			mockServer.Close() // Close the previous mock server
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("POST"))
				Expect(r.URL.Path).To(Equal("/oauth/token"))
				// Simulate an error response
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte(`{"error": "Internal Server Error"}`))
				Expect(err).NotTo(HaveOccurred()) // Check the error from w.Write
			}))
			defer mockServer.Close() // Ensure the mock server is closed after the test
			fakeRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &OAuthTokenConfigReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				EventRecorder: fakeRecorder,        // Use the EventRecorder
				HTTPClient:    mockServer.Client(), // Use the mock HTTP client
			}
			// Attempt to reconcile with the mock server that returns an error
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred(), "Expected an error when token refresh fails")

			found := false
			for len(fakeRecorder.Events) > 0 { // Drain all events from the channel
				event := <-fakeRecorder.Events
				if strings.Contains(event, "TokenRefreshFailed") && strings.Contains(event, "Failed to refresh token") {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Expected 'TokenRefreshFailed' event was not recorded")
			// Verify that no target secret was created
			target := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: targetSecret, Namespace: namespace}, target)
			Expect(err).To(HaveOccurred(), "Expected an error when target secret is not found")
			Expect(target.Name).To(BeEmpty(), "Expected target secret to not be created")
		})

		It("should emit event if credentials secret not found", func() {
			By("Deleting the credentials secret")
			credentials := &corev1.Secret{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: credentialsSecret, Namespace: namespace}, credentials)
			if err == nil {
				Expect(k8sClient.Delete(ctx, credentials)).To(Succeed())
			}
			fakeRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &OAuthTokenConfigReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				EventRecorder: fakeRecorder,        // Use the EventRecorder
				HTTPClient:    mockServer.Client(), // Use the mock HTTP client
			}
			// Attempt to reconcile with the missing credentials secret
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred(), "Expected an error when credentials secret is not found")

			// Verify that the "ResourceFetchFailed" event was recorded
			found := false
			for len(fakeRecorder.Events) > 0 { // Drain all events from the channel
				event := <-fakeRecorder.Events
				if strings.Contains(event, "ResourceFetchFailed") && strings.Contains(event, "CredentialsSecret") {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Expected 'ResourceFetchFailed' event was not recorded")
			// Verify that no target secret was created
			target := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: targetSecret, Namespace: namespace}, target)
			Expect(err).To(HaveOccurred(), "Expected an error when target secret is not found")
			Expect(target.Name).To(BeEmpty(), "Expected target secret to not be created")
		})

		It("should emit event if credentials secret malformed", func() {
			By("Creating a malformed credentials secret")
			// Delete the existing credentials secret if it exists
			credentials := &corev1.Secret{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: credentialsSecret, Namespace: namespace}, credentials)
			if err == nil {
				Expect(k8sClient.Delete(ctx, credentials)).To(Succeed())
			}
			// Create a malformed credentials secret
			malformedCredentials := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      credentialsSecret,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					clientIDField: []byte("test-client-id"),
					// Missing clientSecretField
				},
			}
			Expect(k8sClient.Create(ctx, malformedCredentials)).To(Succeed())

			fakeRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &OAuthTokenConfigReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				EventRecorder: fakeRecorder,        // Use the EventRecorder
				HTTPClient:    mockServer.Client(), // Use the mock HTTP client
			}

			// Attempt to reconcile with the malformed credentials
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred(), "Expected an error when credentials secret is malformed")

			// Verify that the "ResourceValidationFailed" event was recorded
			found := false
			for len(fakeRecorder.Events) > 0 { // Drain all events from the channel
				event := <-fakeRecorder.Events
				if strings.Contains(event, "ResourceValidationFailed") && strings.Contains(event, "Credentials secret validation failed") {
					found = true
					break
				}
			}

			Expect(found).To(BeTrue(), "Expected 'ResourceValidationFailed' event was not recorded")

			// Verify that no target secret was created
			target := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: targetSecret, Namespace: namespace}, target)
			Expect(err).To(HaveOccurred())
			Expect(target.Name).To(BeEmpty())
		})

		It("should emit event if credentials secret malformed for type ropc", func() {
			By("Creating a malformed credentials secret for type ropc")
			// Delete the existing credentials secret if it exists
			credentials := &corev1.Secret{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: credentialsSecret, Namespace: namespace}, credentials)
			if err == nil {
				Expect(k8sClient.Delete(ctx, credentials)).To(Succeed())
			}
			// Create a malformed credentials secret
			malformedCredentials := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      credentialsSecret,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					clientIDField:     []byte("test-client-id"),
					clientSecretField: []byte("test-client-secret"),
					usernameField:     []byte("test-username"),
					// Missing passwordField
				},
			}
			Expect(k8sClient.Create(ctx, malformedCredentials)).To(Succeed())

			fakeRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &OAuthTokenConfigReconciler{
				Client:        k8sClient,
				Scheme:        k8sClient.Scheme(),
				EventRecorder: fakeRecorder,        // Use the EventRecorder
				HTTPClient:    mockServer.Client(), // Use the mock HTTP client
			}

			// Attempt to reconcile with the malformed credentials
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred(), "Expected an error when credentials secret is malformed for type ropc")

			// Verify that the "ResourceValidationFailed" event was recorded
			found := false
			for len(fakeRecorder.Events) > 0 { // Drain all events from the channel
				event := <-fakeRecorder.Events
				if strings.Contains(event, "ResourceValidationFailed") && strings.Contains(event, "Credentials secret validation failed") {
					found = true
					break
				}
			}

			Expect(found).To(BeTrue(), "Expected 'ResourceValidationFailed' event was not recorded")

			// Verify that no target secret was created
			target := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: targetSecret, Namespace: namespace}, target)
			Expect(err).To(HaveOccurred())
			Expect(target.Name).To(BeEmpty())
		})
	})
})
