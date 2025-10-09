# SSOT GraphQL API — Microsoft Entra ID (Azure AD) OIDC Integration

This document provides a complete, end-to-end guide for integrating **Microsoft Entra ID (Azure AD)** authentication into the **SSOT GraphQL API** hosted on AWS Fargate, using **Application Load Balancer (ALB)** with OIDC authentication.

---

## 📘 Overview

The goal is to secure the SSOT GraphQL API and GraphQL Playground by requiring users to sign in via Mavik’s corporate Microsoft accounts.  
Once authenticated, the AWS ALB injects the user’s OIDC token into each request (`x-amzn-oidc-data` header).  
The backend verifies the token, extracts user info, and enforces access control through an ACL stored in AWS Secrets Manager.

### Architecture

User → ALB (OIDC Auth via Entra ID)
→ x-amzn-oidc-data header
→ ECS Fargate (SSOT GQL API)
├─ Verify JWT (issuer, audience, signature, expiry)
└─ Map email → ACL (AWS Secrets Manager)

---

## 1. Azure Entra ID Configuration

### 1.1 Register Application
1. Go to **Azure Portal → Entra ID → App registrations → New registration**
2. Fill in:
   - **Name:** `SSOT-GQL-API`
   - **Supported account types:** `Accounts in this organizational directory only`
   - **Redirect URI:** `https://gql-staging.mavik-ssot.com/oauth2/idpresponse`
3. Click **Register**
4. Record:
   - **Application (client) ID**
   - **Directory (tenant) ID**

### 1.2 Add API Permissions
Go to **API permissions → Add a permission → Microsoft Graph**  
Enable:  openid profile email

### 1.3 Create Client Secret
Go to **Certificates & secrets → New client secret**  
Copy the **Secret Value** — it will be required in ALB setup.

### 1.4 Retrieve OIDC Endpoints
From **App → Endpoints**, note the following:

| Purpose | URL |
|----------|-----|
| **Issuer** | `https://login.microsoftonline.com/<tenant-id>/v2.0` |
| **Authorization Endpoint** | `https://login.microsoftonline.com/<tenant-id>/oauth2/v2.0/authorize` |
| **Token Endpoint** | `https://login.microsoftonline.com/<tenant-id>/oauth2/v2.0/token` |
| **User Info Endpoint** | `https://graph.microsoft.com/oidc/userinfo` |
| **JWKS Keys** | `https://login.microsoftonline.com/<tenant-id>/discovery/v2.0/keys` |

---

## 2. AWS ALB Configuration

### 2.1 Add Authentication Rule
1. Open **EC2 Console → Load Balancers → Listeners → View/Edit rules**
2. Add a new rule **before** forwarding to the target group:
   - **Action:** Authenticate users
   - **Authentication type:** OIDC

### 2.2 Configure OIDC Parameters
| Field | Value |
|--------|--------|
| **Issuer** | `https://login.microsoftonline.com/<tenant-id>/v2.0` |
| **Authorization Endpoint** | same as above |
| **Token Endpoint** | same as above |
| **User Info Endpoint** | same as above |
| **Client ID** | `<application-client-id>` |
| **Client Secret** | `<client-secret>` |
| **Session Cookie Name** | `AWSELBAuthSessionCookie` |
| **Scope** | `openid profile email` |
| **On Unauthenticated Request** | `authenticate` |

Save and apply changes.  
After deployment, visiting `https://gql-staging.mavik-ssot.com/` will redirect users to Microsoft login.  
Once signed in, ALB automatically adds the OIDC token to request headers.

---

## 3. Backend (Golang) Implementation

### 3.1 Retrieve Token
```go
token := r.Header.Get("x-amzn-oidc-data")
```

### 3.2 Verify JWT Signature
Retrieve JWKS from Entra ID and verify the token:
```go
resp, _ := http.Get("https://login.microsoftonline.com/<tenant-id>/discovery/v2.0/keys")
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
// Parse JWK to extract public key and validate signature
jwt.ParseWithClaims(tokenString, &claims, keyFunc)
```

### 3.3 Extract User Information
```go
email := claims["email"].(string)
name  := claims["name"].(string)
```

## 4. Access Control (ACL)

### 4.1 Define ACL in AWS Secrets Manager
Example secret ssot_acl.json (Example, not prod):

```json
{
  "LoanCashFlowReport": ["abc@mavikcapital.com"]
}
```

### 4.2 Implement ACL Check
```go
if allowed, ok := acl[email]; ok && contains(allowed, project) {
    return data
}
return errors.New("unauthorized access")
```

### References

[Microsoft Docs – OIDC Protocol](https://learn.microsoft.com/en-us/azure/active-directory/develop/v2-protocols-oidc)

[AWS Docs – ALB OIDC Authentication](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/listener-authenticate-users.html)

[AWS SDK for Go – JWT Validation](https://pkg.go.dev/github.com/golang-jwt/jwt)
