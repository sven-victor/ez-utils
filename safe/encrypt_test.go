package safe

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	w "github.com/sven-victor/ez-utils/wrapper"
)

func TestEncrypt(t *testing.T) {
	type args struct {
		key string
		o   *EncryptOptions
	}
	tests := []struct {
		name           string
		args           args
		want           string
		encryptWantErr bool
		decryptWantErr bool
	}{{
		name: "none",
		args: args{
			key: "$L9+W9M!jbGMPjKln7Rn6Ge.",
		},
	}, {
		name: "invalid algorithm",
		args: args{
			key: "$L9+W9M!jbGMPjKln7Rn6Ge.",
			o:   NewEncryptOptions(WithAlgorithm(EncryptionAlgorithm(255))),
		},
		decryptWantErr: true,
		encryptWantErr: true,
	}, {
		name: "default",
		args: args{
			key: "$L9+W9M!jbGMPjKln7Rn6Ge.",
			o:   NewEncryptOptions(),
		},
	}, {
		name: "fixAlgo-8",
		args: args{
			key: "$L9+W9M!",
			o:   NewEncryptOptions(WithFixAlgo),
		},
	}, {
		name: "fixAlgo-16",
		args: args{
			key: "$L9+W9M!$L9+W9M!",
			o:   NewEncryptOptions(WithFixAlgo),
		},
	}, {
		name: "fixAlgo-24",
		args: args{
			key: "$L9+W9M!$L9+W9M!$L9+W9M!",
			o:   NewEncryptOptions(WithFixAlgo, WithAlgorithm(AlgorithmDES)),
		},
	}, {
		name: "fixAlgo-32",
		args: args{
			key: "$L9+W9M!$L9+W9M!$L9+W9M!$L9+W9M!",
			o:   NewEncryptOptions(WithFixAlgo),
		},
	}, {
		name: "fixAlgo-31",
		args: args{
			key: "$L9+W9M!$L9+W9M!$L9+W9M!$L9+W9M",
			o:   NewEncryptOptions(WithFixAlgo),
		},
	}, {
		name: "des",
		args: args{
			key: "12345678",
			o:   NewEncryptOptions(WithAlgorithm(AlgorithmDES)),
		},
	}, {
		name: "3des",
		args: args{
			key: "$L9+W9M!jbGMPjKln7Rn6Ge.",
			o:   NewEncryptOptions(WithAlgorithm(Algorithm3DES)),
		},
	}, {
		name: "3des-with_cbc",
		args: args{
			key: "$L9+W9M!jbGMPjKln7Rn6Ge.",
			o:   NewEncryptOptions(WithAlgorithm(Algorithm3DES), WithMode(BlockModeCBC)),
		},
	}, {
		name: "aes",
		args: args{
			key: "$L9+W9M!jbGMPjKln7Rn6Ge.",
			o:   NewEncryptOptions(WithAlgorithm(AlgorithmAES)),
		},
	}, {
		name: "tea",
		args: args{
			key: "1234567890123456",
			o:   NewEncryptOptions(WithAlgorithm(AlgorithmTEA)),
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for dataLength := 1; dataLength < 1024; dataLength++ {
				originalData := make([]byte, dataLength)
				_, err := rand.Read(originalData)
				require.NoError(t, err)
				encryptedStr, err := Encrypt(originalData, tt.args.key, tt.args.o)
				if (err != nil) != tt.encryptWantErr {
					t.Logf("%s => %s", hex.EncodeToString(originalData), encryptedStr)
					t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.encryptWantErr)
					return
				} else if err != nil {
					return
				}
				decryptBytes, err := Decrypt(encryptedStr, tt.args.key)
				if (err != nil) != tt.decryptWantErr {
					t.Logf("%s => %s", hex.EncodeToString(originalData), encryptedStr)
					t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.decryptWantErr)
					return
				} else if err != nil {
					return
				}
				if !bytes.Equal(decryptBytes, originalData) {
					t.Logf("%s => %s", hex.EncodeToString(originalData), encryptedStr)
					t.Errorf("Decrypt(Encrypt()) Do not want to wait with the original data, got = %v, want %v", decryptBytes, originalData)
					return
				}
				if dataLength == 10 {
					t.Logf("%s => %s", base64.StdEncoding.EncodeToString(originalData), encryptedStr)
				}
			}
		})
	}
}

func TestDecrypt(t *testing.T) {
	type args struct {
		cipherString string
		key          string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{{

		name: "3des",
		args: args{
			key:          "$L9+W9M!jbGMPjKln7Rn6Ge.",
			cipherString: "{CRYPT}$0$lh16O4SWHXo7gh24mw$JlTXqqdOPdiTDvqRdKWlNQ",
		},
		want: w.M(base64.StdEncoding.DecodeString("rVmfgMmNxEMeqQ==")),
	}, {
		name: "3des-31",
		args: args{
			key:          "$L9+W9M!$L9+W9M!$L9+W9M!$L9+W9M",
			cipherString: "{CRYPT}$2$cIQUJQ1whBQlC7ZPQEKEWQJMuHD9$x5J/YJhEze9oNe3E7uHi0A",
		},
		want: w.M(base64.StdEncoding.DecodeString("44cDa85fMyrFiQ==")),
	}, {
		name: "cipherString too short",
		args: args{
			key:          "12345678",
			cipherString: "tYYQ",
		},
		wantErr: true,
	}, {
		name: "invalid cipherString",
		args: args{
			key:          "12345678",
			cipherString: "{CRYPT}lh16O4SWHXo7gh24mw$JlTXqqdOPdiTDvqRdKWlNQ",
		},
		wantErr: true,
	}, {
		name: "invalid cipherString2",
		args: args{
			key:          "12345678",
			cipherString: "{CRYPT}$1lh16O4SWHXo7gh24mw$JlTXqqdOPdiTDvqRdKWlNQ",
		},
		wantErr: true,
	}, {
		name: "invalid cipherString3",
		args: args{
			key:          "12345678",
			cipherString: "{CRYPT}$9$lh16O4SWHXo7gh24mw$JlTXqqdOPdiTDvqRdKWlNQ",
		},
		wantErr: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decrypt(tt.args.cipherString, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decrypt() got = %v, want %v", got, tt.want)
			}
		})
	}
}
