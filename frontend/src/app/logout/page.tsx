'use client';

import React from 'react';
import Link from 'next/link';

/**
 * Logout landing page - this page is unauthenticated
 * Users are redirected here after successful logout from the IdP
 * This page should NOT be behind ALB authentication rules
 */
export default function LogoutPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8">
        <div className="text-center">
          <h2 className="mt-6 text-3xl font-extrabold text-gray-900">
            Successfully Logged Out
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            You have been successfully logged out of your account.
          </p>
        </div>
        
        <div className="mt-8 space-y-4">
          <div className="text-center">
            <p className="text-sm text-gray-500 mb-4">
              Your session has been terminated and all authentication tokens have been cleared.
            </p>
            
            <Link
              href="/"
              className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Return to Home
            </Link>
          </div>
          
          <div className="text-center">
            <p className="text-xs text-gray-400">
              If you need to log in again, you will be redirected to the authentication provider.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}