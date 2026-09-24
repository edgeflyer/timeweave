// Package jwt 提供 JWT 签发与解析的封装。
//
// 算法：HS256（HMAC-SHA256），对称密钥，适合单服务部署。
// 如果未来拆微服务需要非对称签名，把这里换成 RS256 / ES256 即可，业务层无需感知。
//
// 安全要点：
//   - 强制只接受 HS256，拒掉 `alg: none` 攻击
//   - 用 jwt.WithValidMethods 而不是手写 alg 判断
//   - 不要在 token 里塞敏感字段（payload 只签名不加密，谁拿到都能 base64 解码）
package jwt

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 算法名常量，方便将来换算法时改一处。
const algorithmHS256 = "HS256"

// 业务层可识别的错误类型，handler 里 errors.Is() 判断后转 HTTP 状态码。
var (
	ErrTokenInvalid = errors.New("jwt: token invalid")
	ErrTokenExpired = errors.New("jwt: token expired")
)

// Claims 是 QuestOS 自己的 token 载荷。
//
// UserID 是业务字段；RegisteredClaims 是 JWT 标准字段（exp / iat / nbf / sub / iss）。
type Claims struct {
	UserID uint64 `json:"user_id"`
	jwt.RegisteredClaims
}

// Issue 用 secret 签发一个 token，userID 作为主体。
//
// ttl：token 有效期，例如 24*time.Hour。
func Issue(secret string, userID uint64, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Subject:   strconv.FormatUint(userID, 10),
			Issuer:    "questos",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

// Parse 解析并校验 token，返回载荷里的 userID。
//
// 校验失败时区分两种：
//   - ErrTokenExpired：过期（前端可以提示用户重新登录）
//   - ErrTokenInvalid：伪造 / 算法不匹配 / 签名错误（401 直接拒）
func Parse(secret, tokenString string) (uint64, error) {
	claims := &Claims{}

	// jwt.WithValidMethods 强制只接受 HS256，挡住 alg=none 和 RS256→HS256 攻击
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(_ *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{algorithmHS256}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, ErrTokenExpired
		}
		return 0, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}
	if !token.Valid {
		return 0, ErrTokenInvalid
	}
	return claims.UserID, nil
}