package aihubsdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

// signLikeServer 用与 server callback_notifier 完全一致的算法生成签名头，用于测试。
func signLikeServer(apiKey string, ts int64, body []byte) (sig, tsStr string) {
	sum := sha256.Sum256([]byte(apiKey))
	secret := hex.EncodeToString(sum[:])
	tsStr = strconv.FormatInt(ts, 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(tsStr + "." + string(body)))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil)), tsStr
}

func TestVerifyCallback_OK(t *testing.T) {
	body := []byte(`{"jobId":"j1","state":"succeeded"}`)
	sig, ts := signLikeServer("mykey", time.Now().Unix(), body)
	if err := VerifyCallback("mykey", sig, ts, body, 5*time.Minute); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestVerifyCallback_BadSignature(t *testing.T) {
	body := []byte(`{"jobId":"j1"}`)
	_, ts := signLikeServer("mykey", time.Now().Unix(), body)
	if err := VerifyCallback("mykey", "sha256=deadbeef", ts, body, 5*time.Minute); err == nil {
		t.Fatal("want signature error, got nil")
	}
}

func TestVerifyCallback_WrongKey(t *testing.T) {
	body := []byte(`{"jobId":"j1"}`)
	sig, ts := signLikeServer("mykey", time.Now().Unix(), body)
	if err := VerifyCallback("otherkey", sig, ts, body, 5*time.Minute); err == nil {
		t.Fatal("want signature error for wrong key, got nil")
	}
}

func TestVerifyCallback_Expired(t *testing.T) {
	body := []byte(`{"jobId":"j1"}`)
	old := time.Now().Add(-10 * time.Minute).Unix()
	sig, ts := signLikeServer("mykey", old, body)
	if err := VerifyCallback("mykey", sig, ts, body, 5*time.Minute); err == nil {
		t.Fatal("want expired error, got nil")
	}
}

func TestVerifyCallback_BadTimestamp(t *testing.T) {
	body := []byte(`{"jobId":"j1"}`)
	sig, _ := signLikeServer("mykey", time.Now().Unix(), body)
	if err := VerifyCallback("mykey", sig, "not-a-number", body, 5*time.Minute); err == nil {
		t.Fatal("want timestamp parse error, got nil")
	}
}
