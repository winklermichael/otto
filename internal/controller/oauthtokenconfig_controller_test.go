package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	authv1alpha1 "github.com/winklermichael/otto/api/v1alpha1"
)

var _ = Describe("OAuthTokenConfig Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			namespace         = "default"
			credentialsSecret = "test-credentials-secret"
			targetSecret      = "test-target-secret"
			clientIDField     = "client_id"
			clientSecretField = "client_secret"
			usernameField     = "username"
			passwordField     = "password"
			refreshTokenField = "refresh_token"
			tokenField        = "access_token"
			refreshURL        = "https://example.com/oauth/token"
		)

		ctx := context.Background()
		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}

		var mockServer *httptest.Server

		BeforeEach(func() {
			// Create a mock HTTP server
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Method).To(Equal("POST"))
				Expect(r.URL.Path).To(Equal("/oauth/token"))

				// Simulate a successful token response
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{
					"access_token": "mock-access-token",
					"refresh_token": "mock-refresh-token",
					"expires_in": 3600
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
					RefreshURL:              mockServer.URL + "/oauth/token", // Use the mock server URL
					CredentialsSecretRef:    corev1.SecretReference{Name: credentialsSecret, Namespace: namespace},
					TargetSecretRef:         corev1.SecretReference{Name: targetSecret, Namespace: namespace},
					Type:                    "ropc",
					TokenFieldName:          tokenField,
					RefreshTokenFieldName:   refreshTokenField,
					ClientIDFieldName:       clientIDField,
					ClientSecretFieldName:   clientSecretField,
					UsernameFieldName:       usernameField,
					PasswordFieldName:       passwordField,
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

		It("should successfully reconcile the resource", func() {
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
			Expect(target.Data).To(HaveKey(tokenField))
			Expect(target.Data).To(HaveKey(refreshTokenField))

			// Verify that the status of the OAuthTokenConfig resource was updated
			oauthTokenConfig := &authv1alpha1.OAuthTokenConfig{}
			err = k8sClient.Get(ctx, typeNamespacedName, oauthTokenConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(oauthTokenConfig.Status.Status).To(Equal("Refreshing"))
			Expect(oauthTokenConfig.Status.LastRefreshed.Time).To(BeTemporally("~", time.Now(), time.Minute))
		})
	})
})
