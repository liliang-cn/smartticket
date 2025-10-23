import { test, expect } from '@playwright/test';
import { authenticate, apiRequest, expectSuccessResponse, expectErrorResponse } from '../helpers/test-helpers';

test.describe('Authentication API', () => {
  const BASE_URL = process.env.BASE_URL || 'http://localhost:6533';

  test('should login with valid credentials', async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectSuccessResponse(response, 200);
    const data = await response.json();

    expect(data.success).toBe(true);
    expect(data.user).toBeDefined();
    expect(data.user.email).toBe('admin@smartticket.local');
    expect(data.user.role).toBe('admin');
    expect(data.tokens).toBeDefined();
    expect(data.tokens.access_token).toBeDefined();
    expect(data.tokens.refresh_token).toBeDefined();
    expect(data.tokens.token_type).toBe('Bearer');
  });

  test('should reject login with invalid email', async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'invalid@example.com',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 401, 'INVALID_CREDENTIALS');
  });

  test('should reject login with invalid password', async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'wrongpassword'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 401, 'INVALID_CREDENTIALS');
  });

  test('should reject login without tenant ID', async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json'
      }
    });

    expectErrorResponse(response, 400, 'MISSING_TENANT_ID');
  });

  test('should reject login with missing email', async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 400, 'VALIDATION_ERROR');
  });

  test('should reject login with missing password', async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 400, 'VALIDATION_ERROR');
  });

  test('should get current user profile with valid token', async ({ request }) => {
    // First login to get token
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const accessToken = loginData.tokens.access_token;

    // Now test /me endpoint
    const response = await request.get(`${BASE_URL}/api/v1/auth/me`, {
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'X-Tenant-ID': '1'
      }
    });

    expectSuccessResponse(response, 200);
    const data = await response.json();

    expect(data.success).toBe(true);
    expect(data.data).toBeDefined();
    expect(data.data.user_id).toBe(1);
    expect(data.data.role).toBe('admin');
    expect(data.data.tenant_id).toBe(1);
  });

  test('should reject /me request without token', async ({ request }) => {
    const response = await request.get(`${BASE_URL}/api/v1/auth/me`, {
      headers: {
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 401, 'UNAUTHORIZED');
  });

  test('should reject /me request with invalid token', async ({ request }) => {
    const response = await request.get(`${BASE_URL}/api/v1/auth/me`, {
      headers: {
        'Authorization': 'Bearer invalid.token.here',
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 401, 'UNAUTHORIZED');
  });

  test('should reject /me request without tenant ID', async ({ request }) => {
    // First login to get token
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const accessToken = loginData.tokens.access_token;

    // Now test /me endpoint without tenant ID
    const response = await request.get(`${BASE_URL}/api/v1/auth/me`, {
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });

    expectErrorResponse(response, 400, 'MISSING_TENANT_ID');
  });

  test('should refresh token with valid refresh token', async ({ request }) => {
    // First login to get tokens
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const refreshToken = loginData.tokens.refresh_token;

    // Now test refresh endpoint
    const response = await request.post(`${BASE_URL}/api/v1/auth/refresh`, {
      data: {
        refresh_token: refreshToken
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectSuccessResponse(response, 200);
    const data = await response.json();

    expect(data.success).toBe(true);
    expect(data.tokens).toBeDefined();
    expect(data.tokens.access_token).toBeDefined();
    expect(data.tokens.refresh_token).toBeDefined();
    expect(data.tokens.token_type).toBe('Bearer');
    // New access token should be different from old one
    expect(data.tokens.access_token).not.toBe(loginData.tokens.access_token);
  });

  test('should reject refresh with invalid refresh token', async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/v1/auth/refresh`, {
      data: {
        refresh_token: 'invalid.refresh.token'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 401, 'UNAUTHORIZED');
  });

  test('should reject refresh without tenant ID', async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/v1/auth/refresh`, {
      data: {
        refresh_token: 'some.refresh.token'
      },
      headers: {
        'Content-Type': 'application/json'
      }
    });

    expectErrorResponse(response, 400, 'MISSING_TENANT_ID');
  });

  test('should logout with valid token', async ({ request }) => {
    // First login to get token
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const accessToken = loginData.tokens.access_token;

    // Now test logout
    const response = await request.post(`${BASE_URL}/api/v1/auth/logout`, {
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'X-Tenant-ID': '1'
      }
    });

    expectSuccessResponse(response, 200);
    const data = await response.json();
    expect(data.success).toBe(true);
    expect(data.message).toBeDefined();
  });

  test('should reject logout without token', async ({ request }) => {
    const response = await request.post(`${BASE_URL}/api/v1/auth/logout`, {
      headers: {
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 401, 'UNAUTHORIZED');
  });

  test('should reject logout without tenant ID', async ({ request }) => {
    // First login to get token
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const accessToken = loginData.tokens.access_token;

    // Now test logout without tenant ID
    const response = await request.post(`${BASE_URL}/api/v1/auth/logout`, {
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });

    expectErrorResponse(response, 400, 'MISSING_TENANT_ID');
  });

  test('should validate token with valid token', async ({ request }) => {
    // First login to get token
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const accessToken = loginData.tokens.access_token;

    // Now test validate endpoint
    const response = await request.get(`${BASE_URL}/api/v1/auth/validate`, {
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'X-Tenant-ID': '1'
      }
    });

    expectSuccessResponse(response, 200);
    const data = await response.json();
    expect(data.success).toBe(true);
    expect(data.valid).toBe(true);
  });

  test('should reject validation with invalid token', async ({ request }) => {
    const response = await request.get(`${BASE_URL}/api/v1/auth/validate`, {
      headers: {
        'Authorization': 'Bearer invalid.token.here',
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 401, 'UNAUTHORIZED');
  });

  test('should change password with valid current password', async ({ request }) => {
    // First login to get token
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const accessToken = loginData.tokens.access_token;

    // Now test change password
    const response = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      data: {
        current_password: 'admin123',
        new_password: 'newpassword123',
        confirm_password: 'newpassword123'
      },
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectSuccessResponse(response, 200);
    const data = await response.json();
    expect(data.success).toBe(true);
    expect(data.message).toBeDefined();

    // Verify can login with new password
    const newLoginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'newpassword123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectSuccessResponse(newLoginResponse, 200);

    // Change password back to original for other tests
    const newLoginData = await newLoginResponse.json();
    const newAccessToken = newLoginData.tokens.access_token;

    await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      data: {
        current_password: 'newpassword123',
        new_password: 'admin123',
        confirm_password: 'admin123'
      },
      headers: {
        'Authorization': `Bearer ${newAccessToken}`,
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });
  });

  test('should reject password change with wrong current password', async ({ request }) => {
    // First login to get token
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const accessToken = loginData.tokens.access_token;

    // Now test change password with wrong current password
    const response = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      data: {
        current_password: 'wrongpassword',
        new_password: 'newpassword123',
        confirm_password: 'newpassword123'
      },
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 400, 'INVALID_CURRENT_PASSWORD');
  });

  test('should reject password change when new passwords do not match', async ({ request }) => {
    // First login to get token
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const accessToken = loginData.tokens.access_token;

    // Now test change password with mismatched passwords
    const response = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      data: {
        current_password: 'admin123',
        new_password: 'newpassword123',
        confirm_password: 'differentpassword123'
      },
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    expectErrorResponse(response, 400, 'PASSWORDS_DO_NOT_MATCH');
  });

  test('should reject password change without tenant ID', async ({ request }) => {
    // First login to get token
    const loginResponse = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: {
        email: 'admin@smartticket.local',
        password: 'admin123'
      },
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': '1'
      }
    });

    const loginData = await loginResponse.json();
    const accessToken = loginData.tokens.access_token;

    // Now test change password without tenant ID
    const response = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      data: {
        current_password: 'admin123',
        new_password: 'newpassword123',
        confirm_password: 'newpassword123'
      },
      headers: {
        'Authorization': `Bearer ${accessToken}`,
        'Content-Type': 'application/json'
      }
    });

    expectErrorResponse(response, 400, 'MISSING_TENANT_ID');
  });
});