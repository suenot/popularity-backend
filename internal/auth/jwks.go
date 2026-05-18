package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSKey is a single RSA key from a JSON Web Key Set.
type JWKSKey struct {
	KTY string `json:"kty"`
	KID string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKS is the document returned by /.well-known/jwks.json.
type JWKS struct {
	Keys []JWKSKey `json:"keys"`
}

// Verifier fetches and caches a remote JWKS, and uses it to validate RS256
// tokens. Cache TTL is one hour; a stale-while-error policy keeps the
// service running through transient JWKS endpoint outages.
type Verifier struct {
	url    string
	issuer string
	ttl    time.Duration
	client *http.Client

	mu       sync.RWMutex
	cache    map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// NewVerifier constructs a Verifier. issuer may be empty to skip iss checks.
func NewVerifier(jwksURL, issuer string) *Verifier {
	return &Verifier{
		url:    jwksURL,
		issuer: issuer,
		ttl:    time.Hour,
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  make(map[string]*rsa.PublicKey),
	}
}

// Parse verifies an RS256 token and returns its claims.
func (v *Verifier) Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("auth: unexpected alg %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("auth: missing kid in header")
		}
		return v.keyFor(kid)
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("auth: invalid token")
	}
	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, fmt.Errorf("auth: unexpected issuer %q", claims.Issuer)
	}
	return claims, nil
}

// keyFor returns the public key for kid, fetching JWKS if cache is stale or
// the kid is unknown.
func (v *Verifier) keyFor(kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	k, ok := v.cache[kid]
	fresh := time.Since(v.fetchedAt) < v.ttl
	v.mu.RUnlock()
	if ok && fresh {
		return k, nil
	}
	if err := v.refresh(); err != nil {
		// Stale-while-error: if we still have a cached key, use it.
		v.mu.RLock()
		k, ok = v.cache[kid]
		v.mu.RUnlock()
		if ok {
			return k, nil
		}
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	k, ok = v.cache[kid]
	if !ok {
		return nil, fmt.Errorf("auth: unknown kid %q", kid)
	}
	return k, nil
}

func (v *Verifier) refresh() error {
	req, err := http.NewRequest(http.MethodGet, v.url, nil)
	if err != nil {
		return fmt.Errorf("auth: build jwks request: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("auth: fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: jwks http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("auth: read jwks: %w", err)
	}
	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("auth: decode jwks: %w", err)
	}
	next := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.KTY != "RSA" {
			continue
		}
		pub, err := jwksToRSA(k)
		if err != nil {
			continue
		}
		next[k.KID] = pub
	}
	v.mu.Lock()
	v.cache = next
	v.fetchedAt = time.Now()
	v.mu.Unlock()
	return nil
}

// jwksToRSA converts a JWKS RSA entry into an *rsa.PublicKey.
func jwksToRSA(k JWKSKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	if e == 0 {
		return nil, errors.New("zero exponent")
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}
