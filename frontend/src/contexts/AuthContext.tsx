'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import { User } from '../lib/auth';

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  login: (oidcData?: string) => void;
  logout: () => void;
  oidcToken: string | null;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [oidcToken, setOidcToken] = useState<string | null>(null);

  useEffect(() => {
    // Check if user is already authenticated
    console.log('Verifying environment variables:');
    console.log('Tenant ID:', process.env.NEXT_PUBLIC_AZURE_AD_TENANT_ID);
    console.log('Post Logout Redirect URI:', process.env.NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI);
    checkAuth();
  }, []);

  const checkAuth = () => {
    setIsLoading(true);
    
    // Skip OIDC authentication in development mode
    if (process.env.NODE_ENV === 'development') {
      // Provide a mock user for local development
      const mockUser: User = {
        id: 'dev-user-123',
        email: 'dev@localhost.com',
        role: 'developer',
        clientId: 'local-dev-client'
      };
      setUser(mockUser);
      setOidcToken('mock-token-for-development');
      setIsLoading(false);
      return;
    }
    
    try {
      fetch('/api/auth/check', {
        method: 'GET',
        credentials: 'include'
      })
      .then(response => {
        return response.json();
      })
      .then(data => {
        if (data.user) {
          setUser(data.user);
          setOidcToken(data.token);
       }
      })
      .catch(error => {
        // This is expected in development without ALB
      })
      .finally(() => {
        setIsLoading(false);
      });
    } catch (error) {
      setIsLoading(false);
    }
  };

  const login = (oidcData?: string) => {
    // In production, this would never be called as ALB handles authentication
    // This method is kept for compatibility but should not be used
  };

  const logout = () => {
    if (process.env.NODE_ENV !== 'development') {
      const tenant = process.env.NEXT_PUBLIC_AZURE_AD_TENANT_ID || 'common';
      const postLogoutRedirectUri = process.env.NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI;

      if (postLogoutRedirectUri) {
        const logoutUrl = `https://login.microsoftonline.com/${tenant}/oauth2/v2.0/logout?post_logout_redirect_uri=${encodeURIComponent(postLogoutRedirectUri)}`;
        window.location.href = logoutUrl;
      } else {
        console.error('Post-logout redirect URI is not configured. Please set NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI.');
      }
    } else {
      // For local development, just clear local state
      setUser(null);
      setOidcToken(null);
    }
  };

  return (
    <AuthContext.Provider value={{ user, isLoading, login, logout, oidcToken }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}