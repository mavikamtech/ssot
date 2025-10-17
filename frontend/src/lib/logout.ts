/**
 * Logout utilities for ALB authentication
 * Implements AWS ALB logout requirements
 */

/**
 * Initiates logout process by clearing ALB authentication cookies
 * and redirecting to the ALB logout endpoint
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
      // Follow the redirect to ALB logout endpoint
      window.location.href = response.url;
    } else {
      // Fallback: redirect directly to ALB logout
      window.location.href = '/oauth2/logout';
    }
  } catch (error) {
    console.error('Error during logout:', error);
    // Fallback: redirect directly to ALB logout
    window.location.href = '/oauth2/logout';
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
 * Gets the ALB logout URL based on environment
 */
export function getALBLogoutURL(): string {
  if (isALBAuthRequired()) {
    return '/oauth2/logout';
  }
  return '/logout'; // Local logout page
}