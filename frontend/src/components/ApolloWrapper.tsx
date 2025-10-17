'use client';

import { ApolloClient, InMemoryCache, ApolloProvider, createHttpLink } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import { useAuth } from '../contexts/AuthContext';

const httpLink = createHttpLink({
  uri: 'https://gql-prod.mavik-ssot.com/query', // Use production API
});

export function ApolloWrapper({ children }: { children: React.ReactNode }) {
  const { oidcToken } = useAuth();

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
      headers: authHeaders
    };
  });

  const client = new ApolloClient({
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

  return (
    <ApolloProvider client={client}>
      {children}
    </ApolloProvider>
  );
}