// Copyright 2026 Sven Victor
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package safe

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"gopkg.in/yaml.v3"
)

// String stores a value that is encrypted at rest using SecretEnvName or an explicit secret.
type String struct {
	value  string
	secret string
}

// Reset clears the encrypted value and secret.
func (e *String) Reset() { *e = String{} }

func (e *String) getSecret() string {
	if e.secret != "" {
		return e.secret
	}
	return os.Getenv(SecretEnvName)
}

// String returns the ciphertext form of the value, encrypting plaintext when a secret is available.
func (e *String) String() string {
	if e.value == "" {
		return ""
	}
	if !strings.HasPrefix(e.value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" {
			if value, err := Encrypt([]byte(e.value), e.secret, nil); err == nil {
				e.value = value
			}
		}
	}
	return e.value
}

// XXX_WellKnownType reports the protobuf well-known type name for String.
func (e *String) XXX_WellKnownType() string { return "StringValue" } //nolint:revive

// ProtoMessage marks String as a protobuf message.
func (e *String) ProtoMessage() {}

// UnmarshalJSONPB implements jsonpb unmarshaling by decoding JSON into String.
func (e *String) UnmarshalJSONPB(_ *jsonpb.Unmarshaler, bytes []byte) error {
	return e.UnmarshalJSON(bytes)
}

// MarshalJSONPB implements jsonpb marshaling by encoding String as JSON.
func (e String) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return e.MarshalJSON()
}

// MarshalJSON implements json.Marshaler and encodes the ciphertext form of the value.
func (e String) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

// UnmarshalJSON implements json.Unmarshaler and encrypts plaintext when a secret is available.
func (e *String) UnmarshalJSON(bytes []byte) (err error) {
	if err = json.Unmarshal(bytes, &e.value); err != nil {
		return err
	}
	if !strings.HasPrefix(e.value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" {
			if e.value, err = Encrypt([]byte(e.value), secret, nil); err != nil {
				return err
			}
			e.secret = secret
		}
	}
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler and encrypts plaintext when a secret is available.
func (e *String) UnmarshalYAML(value *yaml.Node) (err error) {
	if err := value.Decode(&e.value); err != nil {
		return err
	}
	if !strings.HasPrefix(e.value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" {
			if e.value, err = Encrypt([]byte(e.value), secret, nil); err != nil {
				return err
			}
			e.secret = secret
		}
	}
	return nil
}

// UnsafeString decrypts and returns the plaintext value.
func (e String) UnsafeString() (string, error) {
	if e.value == "" {
		return "", nil
	}
	if strings.HasPrefix(e.value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" {
			decrypt, err := Decrypt(e.value, secret)
			return string(decrypt), err
		}
	}
	return e.value, nil
}

// Size returns the length of the stored (possibly encrypted) value.
func (e String) Size() int {
	return len(e.value)
}

// GormDataType reports the GORM column type used to persist String.
func (*String) GormDataType() string {
	return "string"
}

// SetValue stores value, encrypting it when a secret is available and value is plaintext.
func (e *String) SetValue(value string) (err error) {
	if !strings.HasPrefix(value, ciphertextPrefix) {
		if secret := e.getSecret(); secret != "" && value != "" {
			e.secret = secret
			e.value, err = Encrypt([]byte(value), e.secret, nil)
			return err
		}
	}
	e.value = value
	return nil
}

// UpdateSecret re-encrypts the current plaintext with secret and stores that key.
func (e *String) UpdateSecret(secret string) {
	plain, err := e.UnsafeString()
	if err != nil {
		return
	}
	if len(secret) != 0 {
		if safeString, err := Encrypt([]byte(plain), secret, nil); err == nil {
			e.value = safeString
		}
	}
	e.secret = secret
}

// SetSecret records the key used to encrypt and decrypt the value without changing the stored ciphertext.
func (e *String) SetSecret(secret string) {
	e.secret = secret
}

// Scan implements the Scanner interface.
func (e *String) Scan(value any) error {
	switch vt := value.(type) {
	case []uint8:
		e.value = string(vt)
	case string:
		e.value = vt
	default:
		return fmt.Errorf("failed to resolve field, type exception: %T", value)
	}
	return nil
}

// Value implements the driver Valuer interface.
func (e String) Value() (driver.Value, error) {
	return e.value, nil
}

// Equal reports whether e and n have the same plaintext value.
func (e *String) Equal(n String) bool {
	plain, err := e.UnsafeString()
	if err != nil {
		return false
	}
	plain2, err := n.UnsafeString()
	if err != nil {
		return false
	}
	return plain == plain2

}

// NewEncryptedString encrypts plain with secret and returns a String holding the ciphertext.
func NewEncryptedString(plain, secret string) *String {
	if !strings.HasPrefix(plain, ciphertextPrefix) {
		if len(secret) > 0 {
			if safeString, err := Encrypt([]byte(plain), secret, nil); err == nil {
				return &String{value: safeString, secret: secret}
			}
		}
	}
	return &String{value: plain, secret: secret}
}

// SafeStringHookFunc returns a mapstructure.DecodeHookFunc that converts strings into String values.
func SafeStringHookFunc() func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		var dt String
		if t == reflect.TypeOf(dt) {
			var err error
			dt.value = data.(string)
			dt.secret = dt.getSecret()
			if !strings.HasPrefix(dt.value, ciphertextPrefix) {
				if secret := dt.getSecret(); secret != "" {
					if dt.value, err = Encrypt([]byte(dt.value), secret, nil); err != nil {
						return String{}, err
					}
					dt.secret = secret
				}
			}
			return dt, nil
		}
		return data, nil
	}
}
