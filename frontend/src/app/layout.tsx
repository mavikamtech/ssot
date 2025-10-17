import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { AuthProvider } from '../contexts/AuthContext'
import { ApolloWrapper } from '../components/ApolloWrapper'
import '../lib/startupLogger' // Import to trigger startup logging

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'SSOT Frontend',
  description: 'Single Source of Truth Frontend Application',
}

// Log server startup information
if (typeof window === 'undefined') {
  console.log('🚀 SSOT Frontend Server Starting...', {
    timestamp: new Date().toISOString(),
    environment: process.env.NODE_ENV || 'development',
    nodeVersion: process.version,
    platform: process.platform,
    architecture: process.arch,
    memoryUsage: process.memoryUsage(),
    pid: process.pid,
    uptime: process.uptime(),
    config: {
      gqlEndpoint: process.env.NEXT_PUBLIC_GQL_ENDPOINT || 'not-set',
      tenantId: process.env.NEXT_PUBLIC_AZURE_AD_TENANT_ID || 'not-set',
      postLogoutUri: process.env.NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI || 'not-set'
    }
  });
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <AuthProvider>
          <ApolloWrapper>
            {children}
          </ApolloWrapper>
        </AuthProvider>
      </body>
    </html>
  )
}