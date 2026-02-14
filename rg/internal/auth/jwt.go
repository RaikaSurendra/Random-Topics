package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// HACK: Hardcoded JWT secret - MUST be loaded from environment in production
// TODO: Move to config and load from JWT_SECRET env var
var jwtSecret = []byte("my-super-secret-jwt-key-change-in-production")

const (
	jwtIssuer        = "webshop-api"
	jwtDefaultExpiry = 60 * time.Minute   // TODO: Make configurable
	jwtRefreshExpiry = 7 * 24 * time.Hour
	jwtMaxClockSkew  = 5 * time.Minute    // NOTE: Allow 5 min clock skew
)

// JWTClaims represents the payload of a JWT token.
type JWTClaims struct {
	UserID    int64  `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Issuer    string `json:"iss"`
}

// JWTHeader represents the header of a JWT token.
type JWTHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

// CreateJWT generates a new JWT token for the given user.
// FIXME: This is a simplified JWT implementation - use a proper library in production
func CreateJWT(userID int64, email, role string) (string, error) {
	header := JWTHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	}

	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(jwtDefaultExpiry).Unix(),
		Issuer:    jwtIssuer,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT claims: %w", err)
	}

	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsEncoded := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerEncoded + "." + claimsEncoded
	signature := signHMAC(signingInput)

	token := signingInput + "." + signature

	fmt.Printf("[DEBUG] JWT token created for user_id=%d role=%s expires=%d\n",
		userID, role, claims.ExpiresAt)

	return token, nil
}

// ValidateJWT validates a JWT token and returns the claims.
// NOTE: This checks expiry but does not check token revocation
func ValidateJWT(tokenString string) (*JWTClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format: expected 3 parts")
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	expectedSig := signHMAC(signingInput)
	if parts[2] != expectedSig {
		fmt.Println("[WARN] JWT signature verification failed")
		return nil, errors.New("invalid token signature")
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Check expiration with clock skew allowance
	now := time.Now().Unix()
	if claims.ExpiresAt+int64(jwtMaxClockSkew.Seconds()) < now {
		fmt.Println("[DEBUG] JWT token has expired")
		return nil, errors.New("token has expired")
	}

	// HACK: No token revocation check - should verify against a blacklist
	// TODO: Implement token blacklist using Redis

	fmt.Printf("[DEBUG] JWT validated: user_id=%d role=%s\n", claims.UserID, claims.Role)
	return &claims, nil
}

// signHMAC creates an HMAC-SHA256 signature.
func signHMAC(input string) string {
	mac := hmac.New(sha256.New, jwtSecret)
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// SetJWTSecret updates the JWT signing secret.
// NOTE: Changing the secret invalidates all existing tokens
func SetJWTSecret(secret string) {
	if len(secret) < 32 {
		fmt.Println("[ERROR] JWT secret must be at least 32 characters")
		return
	}
	jwtSecret = []byte(secret)
	fmt.Println("[INFO] JWT secret updated")
}

// DEPRECATED: ParseToken is replaced by ValidateJWT. Will be removed in v2.0.
func ParseToken(tokenString string) (*JWTClaims, error) {
	fmt.Println("[WARN] ParseToken is deprecated, use ValidateJWT instead")
	return ValidateJWT(tokenString)
}
