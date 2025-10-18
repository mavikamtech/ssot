import { useCallback } from 'react';

export interface AccessLogParams {
  action: 'GET_LOANS' | 'DOWNLOAD_CSV' | 'UPLOAD_CSV';
  route: string;
  method?: string;
  description?: string;
}

export function useAccessLogger() {
  const logAccess = useCallback(async (params: AccessLogParams) => {
    try {
      const response = await fetch('/api/logging/access', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(params),
      });

      if (!response.ok) {
        const errorData = await response.json();
        console.error('Failed to log access:', errorData);
        return false;
      }

      const result = await response.json();
      console.log('Access logged:', result);
      return result.logged;
    } catch (error) {
      console.error('Error logging access:', error);
      return false;
    }
  }, []);

  return { logAccess };
}