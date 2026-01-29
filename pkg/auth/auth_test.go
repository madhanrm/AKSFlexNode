package auth

import (
	"context"
	"testing"

	"go.goms.io/aks/AKSFlexNode/pkg/config"
)

func TestNewAuthProvider(t *testing.T) {
	provider := NewAuthProvider()
	if provider == nil {
		t.Error("NewAuthProvider should not return nil")
	}
}

func TestArcCredential(t *testing.T) {
	provider := NewAuthProvider()
	
	// Note: This will fail if not running in an Arc-enabled environment
	// We're testing that it returns a credential object, not that it works
	_, err := provider.ArcCredential()
	
	// We expect an error in test environment (no Arc MSI available)
	// Just verify the method doesn't panic
	if err == nil {
		t.Log("Arc credential created successfully (unexpected in test environment)")
	} else {
		t.Logf("Arc credential creation failed as expected in test environment: %v", err)
	}
}

func TestServiceCredential(t *testing.T) {
	provider := NewAuthProvider()
	
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid service principal config",
			cfg: &config.Config{
				Azure: config.AzureConfig{
					ServicePrincipal: &config.ServicePrincipalConfig{
						TenantID:     "test-tenant-id",
						ClientID:     "test-client-id",
						ClientSecret: "test-secret",
					},
				},
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred, err := provider.serviceCredential(tt.cfg)
			
			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if !tt.wantErr && cred == nil {
				t.Error("Credential should not be nil")
			}
		})
	}
}

func TestCLICredential(t *testing.T) {
	provider := NewAuthProvider()
	
	// Note: This will fail if Azure CLI is not installed/configured
	// We're testing that it doesn't panic
	_, err := provider.cliCredential()
	
	// We expect an error in environments without Azure CLI configured
	if err == nil {
		t.Log("CLI credential created successfully")
	} else {
		t.Logf("CLI credential creation failed (may be expected): %v", err)
	}
}

func TestUserCredential(t *testing.T) {
	provider := NewAuthProvider()
	
	tests := []struct {
		name    string
		cfg     *config.Config
		useSP   bool
	}{
		{
			name: "with service principal",
			cfg: &config.Config{
				Azure: config.AzureConfig{
					ServicePrincipal: &config.ServicePrincipalConfig{
						TenantID:     "test-tenant-id",
						ClientID:     "test-client-id",
						ClientSecret: "test-secret",
					},
				},
			},
			useSP: true,
		},
		{
			name: "without service principal (fallback to CLI)",
			cfg: &config.Config{
				Azure: config.AzureConfig{
					ServicePrincipal: nil,
				},
			},
			useSP: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred, err := provider.UserCredential(tt.cfg)
			
			// Don't fail on error - environment may not have Azure CLI
			if err != nil {
				t.Logf("UserCredential returned error (may be expected): %v", err)
				return
			}
			
			if cred == nil {
				t.Error("Credential should not be nil when no error")
			}
		})
	}
}

func TestGetAccessToken(t *testing.T) {
	provider := NewAuthProvider()
	
	// Create a service principal credential (will fail to get token without valid creds)
	cfg := &config.Config{
		Azure: config.AzureConfig{
			ServicePrincipal: &config.ServicePrincipalConfig{
				TenantID:     "test-tenant-id",
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
			},
		},
	}
	
	cred, err := provider.serviceCredential(cfg)
	if err != nil {
		t.Fatalf("Failed to create credential: %v", err)
	}
	
	ctx := context.Background()
	
	// This will fail with invalid credentials, but shouldn't panic
	_, err = provider.GetAccessToken(ctx, cred)
	if err == nil {
		t.Error("Expected error with test credentials")
	} else {
		t.Logf("GetAccessToken failed as expected with test credentials: %v", err)
	}
}

func TestGetAccessTokenForResource(t *testing.T) {
	provider := NewAuthProvider()
	
	// Create a service principal credential
	cfg := &config.Config{
		Azure: config.AzureConfig{
			ServicePrincipal: &config.ServicePrincipalConfig{
				TenantID:     "test-tenant-id",
				ClientID:     "test-client-id",
				ClientSecret: "test-secret",
			},
		},
	}
	
	cred, err := provider.serviceCredential(cfg)
	if err != nil {
		t.Fatalf("Failed to create credential: %v", err)
	}
	
	ctx := context.Background()
	
	tests := []struct {
		name     string
		resource string
	}{
		{
			name:     "ARM resource",
			resource: "https://management.azure.com/.default",
		},
		{
			name:     "Graph resource",
			resource: "https://graph.microsoft.com/.default",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will fail with invalid credentials
			_, err := provider.GetAccessTokenForResource(ctx, cred, tt.resource)
			if err == nil {
				t.Error("Expected error with test credentials")
			} else {
				t.Logf("GetAccessTokenForResource failed as expected: %v", err)
			}
		})
	}
}

func TestCheckCLIAuthStatus(t *testing.T) {
	provider := NewAuthProvider()
	ctx := context.Background()
	
	// This will fail if Azure CLI is not installed or user not logged in
	err := provider.CheckCLIAuthStatus(ctx)
	
	if err == nil {
		t.Log("CLI auth status check passed (user is logged in)")
	} else {
		t.Logf("CLI auth status check failed (expected if not logged in): %v", err)
	}
}

// Note: We don't test InteractiveAzLogin and EnsureAuthenticated as they require user interaction
// These should be tested manually or with integration tests
