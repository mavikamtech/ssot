import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { AuthProvider } from '../contexts/AuthContext'
import { ApolloWrapper } from '../components/ApolloWrapper'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'SSOT Frontend',
  description: 'Single Source of Truth Frontend Application',
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