'use client';

import { useMemo } from 'react';
import { ApolloClient, InMemoryCache, ApolloProvider, createHttpLink } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import { useAuth } from '../contexts/AuthContext';

const httpLink = createHttpLink({
  uri: process.env.NEXT_PUBLIC_GQL_ENDPOINT || 'https://gql-prod.mavik-ssot.com/query', // Use env or fallback to production API
});

export function ApolloWrapper({ children }: { children: React.ReactNode }) {
  const { oidcToken } = useAuth();

  const client = useMemo(() => {
    const authLink = setContext((_, { headers }) => {
      const authHeaders: Record<string, string> = {
        ...headers,
        'Content-Type': 'application/json',
      };

      if (oidcToken) {
        authHeaders['x-amzn-oidc-data'] = oidcToken;
        authHeaders['Authorization'] = oidcToken; // This is by design for the backend to read
      }

      return {
        headers: authHeaders,
      };
    });

    return new ApolloClient({
      link: authLink.concat(httpLink),
      cache: new InMemoryCache(),
      defaultOptions: {
        watchQuery: {
          errorPolicy: 'all',
        },
        query: {
          errorPolicy: 'all',
        },
      },
    });
  }, [oidcToken]);

  return <ApolloProvider client={client}>{children}</ApolloProvider>;
}