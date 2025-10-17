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
    checkAuth();
  }, []);

  const checkAuth = () => {
    setIsLoading(true);
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
    setUser(null);
    setOidcToken(null);
    // In production with ALB OIDC, logout would redirect to OIDC provider logout endpoint
    // For now, just clear local state
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