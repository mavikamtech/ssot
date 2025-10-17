/**
 * Logout utilities for ALB authentication
 * Implements AWS ALB logout requirements
 */

/**
 * Initiates logout process by clearing ALB authentication cookies
 * and redirecting to the Azure AD logout endpoint
 */
export async function initiateLogout(): Promise<void> {
  if (typeof window === 'undefined') {
    throw new Error('Logout can only be initiated from the browser');
  }

  if (process.env.NODE_ENV === 'development') {
    // In development, just redirect to home page
    window.location.href = '/';
    return;
  }

  try {
    // Call the logout API endpoint
    const response = await fetch('/api/auth/logout', {
      method: 'POST',
      credentials: 'include',
    });

    if (response.redirected) {
      // Follow the redirect to Azure AD logout endpoint
      window.location.href = response.url;
    } else {
      // Fallback: redirect directly to Azure AD logout
      const azureLogoutUrl = getAzureADLogoutURL();
      window.location.href = azureLogoutUrl;
    }
  } catch (error) {
    console.error('Error during logout:', error);
    // Fallback: redirect directly to Azure AD logout
    const azureLogoutUrl = getAzureADLogoutURL();
    window.location.href = azureLogoutUrl;
  }
}

/**
 * Clears client-side authentication state
 * This should be called when logout is detected
 */
export function clearAuthState(): void {
  // Clear any client-side storage
  if (typeof window !== 'undefined') {
    // Clear localStorage
    localStorage.removeItem('auth-token');
    localStorage.removeItem('user-data');
    
    // Clear sessionStorage  
    sessionStorage.removeItem('auth-token');
    sessionStorage.removeItem('user-data');
  }
}

/**
 * Checks if the current environment requires ALB authentication
 */
export function isALBAuthRequired(): boolean {
  return process.env.NODE_ENV === 'production';
}

/**
 * Gets the Azure AD logout URL based on environment configuration
 */
export function getAzureADLogoutURL(): string {
  if (process.env.NODE_ENV === 'development') {
    return '/logout'; // Local logout page
  }

  const tenant = process.env.NEXT_PUBLIC_AZURE_AD_TENANT_ID || 'common';
  const postLogoutRedirectUri = process.env.NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI;

  if (postLogoutRedirectUri) {
    return `https://login.microsoftonline.com/${tenant}/oauth2/v2.0/logout?post_logout_redirect_uri=${encodeURIComponent(postLogoutRedirectUri)}`;
  }

  // Fallback to basic Azure AD logout without redirect
  return `https://login.microsoftonline.com/${tenant}/oauth2/v2.0/logout`;
}

/**
 * Gets the ALB logout URL based on environment (deprecated - use getAzureADLogoutURL)
 */
export function getALBLogoutURL(): string {
  if (isALBAuthRequired()) {
    return '/oauth2/logout';
  }
  return '/logout'; // Local logout page
}