'use client';

import { useAuth } from '../contexts/AuthContext';
import LoginPage from '../components/LoginPage';
import Dashboard from '../components/Dashboard';
import { useEffect } from 'react';

export default function Home() {
  const { user, isLoading } = useAuth();

  // Add logging and health indicator
  useEffect(() => {
    // Set a data attribute that can be checked by health monitors
    document.documentElement.setAttribute('data-health-status', 'ready');
    
    return () => {
      document.documentElement.removeAttribute('data-health-status');
    };
  }, [user, isLoading]);

  if (isLoading) {
    return (
      <div className="loading">
        <div>Loading...</div>
      </div>
    );
  }

  if (!user) {
    return <LoginPage />;
  }

  return <Dashboard />;
}