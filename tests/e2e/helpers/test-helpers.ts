import { test as base, expect } from '@playwright/test';

// Test data interfaces
interface TestUser {
  id?: number;
  email: string;
  password: string;
  username?: string;
  first_name?: string;
  last_name?: string;
  role: string;
  tenant_id?: number;
}

interface TestTenant {
  id?: number;
  name: string;
  slug: string;
  domain?: string;
}

interface AuthTokens {
  access_token: string;
  refresh_token: string;
  user: TestUser;
}

// Test fixtures
export const test = base.extend({
  // Authenticated requests for different user roles
  adminAuth: async ({ request }, use) => {
    const tokens = await authenticate(request, 'admin', 'admin123', '1');
    await use(tokens);
  },

  tenantAdminAuth: async ({ request }, use) => {
    const tokens = await authenticate(request, 'tenantadmin', 'tenantadmin123', '1');
    await use(tokens);
  },

  customerAuth: async ({ request }, use) => {
    const tokens = await authenticate(request, 'customer', 'customer123', '1');
    await use(tokens);
  },
});

// Constants
const BASE_URL = process.env.BASE_URL || 'http://localhost:6533';

// Helper functions
export async function authenticate(
  request: any,
  email: string,
  password: string,
  tenantId: string
): Promise<AuthTokens> {
  const response = await request.post(`${BASE_URL}/api/v1/auth/login`, {
    data: {
      email,
      password
    },
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId
    }
  });

  if (response.status() !== 200) {
    throw new Error(`Authentication failed for ${email}: ${response.status()}`);
  }

  const data = await response.json();
  if (!data.success) {
    throw new Error(`Authentication failed for ${email}: ${data.error?.message}`);
  }

  return data.data;
}

export async function createTestUser(
  request: any,
  user: Partial<TestUser>,
  authToken: string,
  tenantId: string = '1'
): Promise<TestUser> {
  const defaultUser = {
    email: `test${Date.now()}@example.com`,
    password: 'testpassword123',
    username: `testuser${Date.now()}`,
    first_name: 'Test',
    last_name: 'User',
    role: 'customer',
    ...user
  };

  const response = await request.post(`${BASE_URL}/api/v1/users`, {
    data: defaultUser,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${authToken}`,
      'X-Tenant-ID': tenantId
    }
  });

  if (response.status() !== 201) {
    throw new Error(`Failed to create test user: ${response.status()}`);
  }

  const data = await response.json();
  return data.data;
}

export async function createTestTenant(
  request: any,
  tenant: Partial<TestTenant>,
  authToken: string
): Promise<TestTenant> {
  const defaultTenant = {
    name: `Test Tenant ${Date.now()}`,
    slug: `test-tenant-${Date.now()}`,
    domain: 'test.example.com',
    ...tenant
  };

  const response = await request.post(`${BASE_URL}/api/v1/admin/tenants`, {
    data: defaultTenant,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${authToken}`,
      'X-Tenant-ID': '1'
    }
  });

  if (response.status() !== 201) {
    throw new Error(`Failed to create test tenant: ${response.status()}`);
  }

  const data = await response.json();
  return data.data;
}

export async function cleanupTestData(
  request: any,
  authToken: string,
  tenantId: string = '1'
): Promise<void> {
  // Note: In a real implementation, you'd want to clean up created test data
  // This is a placeholder for cleanup logic
  console.log('Test cleanup completed');
}

// Common test data
export const TEST_USERS = {
  admin: {
    email: 'admin@smartticket.local',
    password: 'admin123',
    role: 'admin'
  },
  tenantAdmin: {
    email: 'tenantadmin@smartticket.local',
    password: 'tenantadmin123',
    role: 'tenant_admin'
  },
  customer: {
    email: 'customer@smartticket.local',
    password: 'customer123',
    role: 'customer'
  }
} as const;

export const TEST_TENANT = {
  name: 'Test Tenant',
  slug: 'test-tenant',
  domain: 'test.smartticket.local'
} as const;

// Response validation helpers
export function expectSuccessResponse(response: any, expectedStatus: number = 200) {
  expect(response.status()).toBe(expectedStatus);
}

export function expectErrorResponse(response: any, expectedStatus: number, expectedCode?: string) {
  expect(response.status()).toBe(expectedStatus);

  if (expectedCode) {
    const data = response.json();
    expect(data.success).toBe(false);
    expect(data.error.code).toBe(expectedCode);
  }
}

export function expectPaginatedResponse(response: any) {
  expectSuccessResponse(response);
  const data = response.json();
  expect(data).toHaveProperty('data');
  expect(data.data).toHaveProperty('items');
  expect(data.data).toHaveProperty('total');
  expect(data.data).toHaveProperty('page');
  expect(data.data).toHaveProperty('page_size');
}

// API request helper
export function apiRequest(request: any, authToken: string, tenantId: string = '1') {
  return {
    get: (url: string, options?: any) =>
      request.get(`${BASE_URL}${url}`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'X-Tenant-ID': tenantId,
          ...options?.headers
        },
        ...options
      }),

    post: (url: string, data?: any, options?: any) =>
      request.post(`${BASE_URL}${url}`, {
        data,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authToken}`,
          'X-Tenant-ID': tenantId,
          ...options?.headers
        },
        ...options
      }),

    put: (url: string, data?: any, options?: any) =>
      request.put(`${BASE_URL}${url}`, {
        data,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authToken}`,
          'X-Tenant-ID': tenantId,
          ...options?.headers
        },
        ...options
      }),

    delete: (url: string, options?: any) =>
      request.delete(`${BASE_URL}${url}`, {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'X-Tenant-ID': tenantId,
          ...options?.headers
        },
        ...options
      })
  };
}

export { expect };