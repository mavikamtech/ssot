'use client';

import React from 'react';
import { useAuth } from '@/contexts/AuthContext';

/**
 * Logout Button Component
 * Demonstrates proper usage of the Azure AD logout functionality
 */
export function LogoutButton({ 
  className = '',
  children = 'Logout'
}: {
  className?: string;
  children?: React.ReactNode;
}) {
  const { logout, user, isLoading } = useAuth();

  const handleLogout = async () => {
    try {
      await logout();
    } catch (error) {
      console.error('Logout failed:', error);
      // Handle error - maybe show a toast notification
    }
  };

  // Don't show logout button if not authenticated or still loading
  if (!user || isLoading) {
    return null;
  }

  return (
    <button
      onClick={handleLogout}
      className={`px-4 py-2 text-sm font-medium text-white bg-red-600 border border-transparent rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50 disabled:cursor-not-allowed ${className}`}
      type="button"
    >
      {children}
    </button>
  );
}

/**
 * User Menu Component with Logout
 * Example of integrating logout in a user dropdown menu
 */
export function UserMenu() {
  const { user, logout } = useAuth();

  if (!user) return null;

  return (
    <div className="relative">
      <div className="flex items-center space-x-4">
        <span className="text-sm text-gray-700">
          Welcome, {user.email}
        </span>
        <LogoutButton className="text-xs px-3 py-1">
          Sign Out
        </LogoutButton>
      </div>
    </div>
  );
}