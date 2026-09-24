// Package password 提供密码哈希与校验的封装。
//
// 业务代码**不应该**直接调用 bcrypt，必须通过本包的 Hash / Check 函数。
// 这样将来如果切换哈希算法（比如改 argon2id 或 scrypt），只改本包一行 import 即可。
//
// bcrypt 限制说明：
//   - 输出固定 60 字节（盐 + 哈希）
//   - **输入最长 72 字节**，超出部分会被静默截断，所以务必在业务层校验密码长度
//   - Cost 默认 10 ≈ 60ms/次；Phase 1 够用，量上来后可调到 12
package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Cost 是 bcrypt 的计算成本因子（2^Cost 次 key schedule 轮）。
const Cost = 10

// ErrTooLong 用户输入的明文超过 72 字节限制。
var ErrTooLong = errors.New("password: exceeds bcrypt 72-byte limit")

// Hash 用 bcrypt 把明文密码哈希成不可逆字符串。
//
// 调用方应该：传入前校验 len(plain) > 0 且 < 72，否则返回 ErrTooLong。
func Hash(plain string) (string, error) {
	if len(plain) > 72 {
		return "", ErrTooLong
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), Cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt generate: %w", err)
	}
	return string(h), nil
}

// Check 比对明文密码和已存的哈希值。密码正确返回 nil，否则返回 error。
func Check(hash, plain string) error {
	if len(plain) > 72 {
		return ErrTooLong
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}