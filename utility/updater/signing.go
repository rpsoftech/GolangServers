package updater

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
)

// ReleasePublicKey is the base64 ed25519 public key that OTA releases must be
// signed with. It is not a secret. Generate a key pair with:
//
//	go run ./utility/deploy keygen
//
// then paste the public key here and keep the private key in the deploy
// machine's UPDATE_SIGNING_KEY environment variable. While this is empty the
// updater refuses every update and the deploy tool refuses to publish.
const ReleasePublicKey = "0Uu99qmXD7yROUkFxLjmcO/0Ml1biyyuH28Zk0YzczI="

// releasePublicKeyB64 exists so tests can substitute a key.
var releasePublicKeyB64 = ReleasePublicKey

var ErrReleaseKeyNotSet = errors.New("updater.ReleasePublicKey is not set; refusing unsigned updates")

// SignedMessage is the exact byte string a release signature covers. Binding
// the KV key and version stops a valid signature for one component,
// platform or environment from being replayed for another, and stops old
// releases from being re-served under a new version number.
func SignedMessage(kvKey string, version int, sha256Hex string) []byte {
	return fmt.Appendf(nil, "rpsoftech-ota-v1\n%s\n%d\n%s", kvKey, version, sha256Hex)
}

func releasePublicKey() (ed25519.PublicKey, error) {
	if releasePublicKeyB64 == "" {
		return nil, ErrReleaseKeyNotSet
	}
	raw, err := base64.StdEncoding.DecodeString(releasePublicKeyB64)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("updater.ReleasePublicKey is not a valid base64 ed25519 public key")
	}
	return ed25519.PublicKey(raw), nil
}

// VerifyRelease checks the signature on a KV release entry.
func VerifyRelease(kvKey string, kvData KVResponse) error {
	pub, err := releasePublicKey()
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(kvData.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("release has no valid signature")
	}
	if !ed25519.Verify(pub, SignedMessage(kvKey, kvData.Version, kvData.SHA256), sig) {
		return errors.New("release signature does not verify")
	}
	return nil
}

// ParseSigningKey decodes a base64 ed25519 private key and checks that it
// matches ReleasePublicKey, so the deploy tool cannot publish releases that
// the shipped updater would reject.
func ParseSigningKey(b64 string) (ed25519.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return nil, errors.New("signing key is not a valid base64 ed25519 private key")
	}
	priv := ed25519.PrivateKey(raw)
	pub, err := releasePublicKey()
	if err != nil {
		return nil, err
	}
	if !pub.Equal(priv.Public()) {
		return nil, errors.New("signing key does not match updater.ReleasePublicKey")
	}
	return priv, nil
}
