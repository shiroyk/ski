package crypto

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/shiroyk/cloudcat/js/modulestest"
	"github.com/stretchr/testify/assert"
)

type MockReader struct{}

func (MockReader) Read(_ []byte) (n int, err error) {
	return -1, errors.New("contrived failure")
}

func TestHashAlgorithms(t *testing.T) {
	if testing.Short() {
		return
	}

	vm := modulestest.New(t)
	_, _ = vm.RunString(context.Background(), `
		const crypto = require('cloudcat/crypto');
	`)

	t.Run("RandomBytesSuccess", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		let buf = crypto.randomBytes(5);
		assert.equal(5, buf.byteLength);
		`)

		assert.NoError(t, err)
	})

	t.Run("RandomBytesInvalidSize", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `crypto.randomBytes(-1);`)

		assert.Error(t, err)
	})

	t.Run("RandomBytesFailure", func(t *testing.T) {
		SavedReader := rand.Reader
		rand.Reader = MockReader{}
		_, err := vm.RunString(context.Background(), `crypto.randomBytes(5);`)
		rand.Reader = SavedReader

		assert.Error(t, err)
	})

	t.Run("MD4", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var correct = "aa010fbc1d14c795d86ef98c95479d17";
		var hash = crypto.md4("hello world").hex();
		assert.equal(correct, hash);
		`)
		assert.NoError(t, err)
	})

	t.Run("MD5", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var correct = "5eb63bbbe01eeed093cb22bb8f5acdc3";
		var hash = crypto.md5("hello world").hex();
		assert.equal(correct, hash);
		`)

		assert.NoError(t, err)
	})

	t.Run("SHA1", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var correct = "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed";
		var hash = crypto.sha1("hello world").hex();
		assert.equal(correct, hash);
		`)

		assert.NoError(t, err)
	})

	t.Run("SHA256", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var correct = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9";
		var hash = crypto.sha256("hello world").hex();
		assert.equal(correct, hash);
		`)

		assert.NoError(t, err)
	})

	t.Run("SHA384", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var correct = "fdbd8e75a67f29f701a4e040385e2e23986303ea10239211af907fcbb83578b3e417cb71ce646efd0819dd8c088de1bd";
		var hash = crypto.sha384("hello world").hex();
		assert.equal(correct, hash);
		`)

		assert.NoError(t, err)
	})

	t.Run("SHA512", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var correct = "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f";
		var hash = crypto.sha512("hello world").hex();
		assert.equal(correct, hash);
		`)

		assert.NoError(t, err)
	})

	t.Run("SHA512_224", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var hash = crypto.sha512_224("hello world").hex();
		var correct = "22e0d52336f64a998085078b05a6e37b26f8120f43bf4db4c43a64ee";
		assert.equal(correct, hash);
		`)

		assert.NoError(t, err)
	})

	t.Run("SHA512_256", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var hash = crypto.sha512_256("hello world").hex();
		var correct = "0ac561fac838104e3f2e4ad107b4bee3e938bf15f2b15f009ccccd61a913f017";
		assert.equal(correct, hash);
		`)

		assert.NoError(t, err)
	})

	t.Run("RIPEMD160", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var hash = crypto.ripemd160("hello world").hex();
		var correct = "98c615784ccb5fe5936fbc0cbe9dfdb408d92f0f";
		assert.equal(correct, hash);
		`)

		assert.NoError(t, err)
	})
}

func TestStreamingApi(t *testing.T) {
	if testing.Short() {
		return
	}

	vm := modulestest.New(t)
	_, _ = vm.RunString(context.Background(), `
		const crypto = require('cloudcat/crypto');
	`)

	// Empty strings are still hashable
	t.Run("Empty", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var correctHex = "d41d8cd98f00b204e9800998ecf8427e";
		var hasher = crypto.createHash("md5");
		assert.equal(correctHex, hasher.digest().hex());
		`)

		assert.NoError(t, err)
	})

	t.Run("UpdateOnce", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var correctHex = "5eb63bbbe01eeed093cb22bb8f5acdc3";

		var hasher = crypto.createHash("md5");
		hasher.update("hello world");
		assert.equal(correctHex, hasher.digest().hex());
		`)

		assert.NoError(t, err)
	})

	t.Run("UpdateMultiple", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		var correctHex = "5eb63bbbe01eeed093cb22bb8f5acdc3";

		var hasher = crypto.createHash("md5");
		hasher.update("hello");
		hasher.update(" ");
		hasher.update("world");

		assert.equal(correctHex, hasher.digest().hex());
		`)

		assert.NoError(t, err)
	})
}

func TestOutputEncoding(t *testing.T) {
	if testing.Short() {
		return
	}

	vm := modulestest.New(t)
	_, _ = vm.RunString(context.Background(), `
		const crypto = require('cloudcat/crypto');
	`)

	t.Run("Valid", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		let correctHex = "5eb63bbbe01eeed093cb22bb8f5acdc3";
		let correctBase64 = "XrY7u+Ae7tCTyyK7j1rNww==";
		let correctBase64URL = "XrY7u-Ae7tCTyyK7j1rNww=="
		let correctBase64RawURL = "XrY7u-Ae7tCTyyK7j1rNww";
		let correctBinary = new Uint8Array([94,182,59,187,224,30,238,208,147,203,34,187,143,90,205,195]).buffer;

		let hasher = crypto.createHash("md5");
		let encoder = hasher.encrypt("hello world");

		assert.equal(correctHex, encoder.hex());
		assert.equal(correctBase64, encoder.base64());
		assert.equal(correctBase64URL, encoder.base64url());
		assert.equal(correctBase64RawURL, encoder.base64rawurl());
		assert.equal(correctBinary, encoder.binary());
		`)

		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		_, err := vm.RunString(context.Background(), `
		crypto.createHash("md5").encrypt("hello world").someInvalidEncoding();
		`)
		assert.Contains(t, err.Error(), "Object has no member 'someInvalidEncoding'")
	})
}

func TestHMac(t *testing.T) {
	if testing.Short() {
		return
	}

	vm := modulestest.New(t)
	_, _ = vm.RunString(context.Background(), `
		const crypto = require('cloudcat/crypto');
	`)

	testData := map[string]string{
		"md4":           "92d8f5c302cf04cca0144d7a9feb1596",
		"md5":           "e04f2ec05c8b12e19e46936b171c9d03",
		"sha1":          "c113b62711ff5d8e8100bbb17b998591af81dc24",
		"sha256":        "7fd04df92f636fd450bc841c9418e5825c17f33ad9c87c518115a45971f7f77e",
		"sha384":        "d331e169e2dcfc742e80a3bf4dcc76d0e6425ab3777a3ac217ac6b2552aad5529ed4d40135b06e53a495ac7425d1e462",
		"sha512_224":    "bac4e6256bdbf81d029aec48af4fdd4b14001db6721f07c429a80817",
		"sha512_256":    "e3d0763ba92a4f40676c3d5b234d9842b71951e6e0767082cfb3f5e14c124b22",
		"sha512":        "cd3146f96a3005024108ff56b025517552435589a4c218411f165da0a368b6f47228b20a1a4bf081e4aae6f07e2790f27194fc77f0addc890e98ce1951cacc9f",
		"ripemd160_256": "00bb4ce0d6afd4c7424c9d01b8a6caa3e749b08b",
	}
	for algorithm, value := range testData {
		_ = vm.Runtime().Set("correctHex", vm.Runtime().ToValue(value))
		_ = vm.Runtime().Set("algorithm", vm.Runtime().ToValue(algorithm))

		t.Run(algorithm+" hasher: valid", func(t *testing.T) {
			_, err := vm.RunString(context.Background(), `
			var hasher = crypto.createHMAC(algorithm, "a secret");
			assert.equal(correctHex, hasher.encrypt("some data to hash").hex());
			`)

			assert.NoError(t, err)
		})

		t.Run(algorithm+" wrapper: valid", func(t *testing.T) {
			_, err := vm.RunString(context.Background(), `
			var resultHex = crypto.hmac(algorithm, "a secret", "some data to hash").hex();
			assert.equal(correctHex, resultHex);
			`)

			assert.NoError(t, err)
		})

		t.Run(algorithm+" ArrayBuffer: valid", func(t *testing.T) {
			_, err := vm.RunString(context.Background(), `
			var data = new Uint8Array([115,111,109,101,32,100,97,116,97,32,116,
										111,32,104,97,115,104]).buffer;
			var resultHex = crypto.hmac(algorithm, "a secret", data).hex();
			assert.equal(correctHex, resultHex);
			`)

			assert.NoError(t, err)
		})
	}

	// Algorithms not supported or typing error
	invalidData := map[string]string{
		"md6":    "e04f2ec05c8b12e19e46936b171c9d03",
		"sha526": "7fd04df92f636fd450bc841c9418e5825c17f33ad9c87c518115a45971f7f77e",
		"sha348": "d331e169e2dcfc742e80a3bf4dcc76d0e6425ab3777a3ac217ac6b2552aad5529ed4d40135b06e53a495ac7425d1e462",
	}
	for algorithm, value := range invalidData {
		algorithm := algorithm
		_ = vm.Runtime().Set("correctHex", vm.Runtime().ToValue(value))
		_ = vm.Runtime().Set("algorithm", vm.Runtime().ToValue(algorithm))
		t.Run(algorithm+" hasher: invalid", func(t *testing.T) {
			_, err := vm.RunString(context.Background(), `
			var hasher = crypto.createHMAC(algorithm, "a secret");	
			assert.equal(correctHex, hasher.hash("some data to hash").hex())
			`)

			assert.Contains(t, err.Error(), "invalid algorithm: "+algorithm)
		})

		t.Run(algorithm+" wrapper: invalid", func(t *testing.T) {
			_, err := vm.RunString(context.Background(), `
			var resultHex = crypto.hmac(algorithm, "a secret", "some data to hash").hex();
			assert.equal(correctHex, resultHex);
			`)

			assert.Contains(t, err.Error(), "invalid algorithm: "+algorithm)
		})
	}
}

func TestAWSv4(t *testing.T) {
	// example values from https://docs.aws.amazon.com/general/latest/gr/signature-v4-examples.html
	vm := modulestest.New(t)

	_, err := vm.RunString(context.Background(), `
		const crypto = require('cloudcat/crypto');
		let hmacSHA256 = function(data, key) {
			return crypto.hmac("sha256", key, data);
		};

		let expectedKDate    = '969fbb94feb542b71ede6f87fe4d5fa29c789342b0f407474670f0c2489e0a0d'
		let expectedKRegion  = '69daa0209cd9c5ff5c8ced464a696fd4252e981430b10e3d3fd8e2f197d7a70c'
		let expectedKService = 'f72cfd46f26bc4643f06a11eabb6c0ba18780c19a8da0c31ace671265e3c87fa'
		let expectedKSigning = 'f4780e2d9f65fa895f9c67b32ce1baf0b0d8a43505a000a1a9e090d414db404d'

		let key = 'wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY';
		let dateStamp = '20120215';
		let regionName = 'us-east-1';
		let serviceName = 'iam';

		let kDate = hmacSHA256(dateStamp, "AWS4" + key);
		let kRegion = hmacSHA256(regionName, kDate.binary());
		let kService = hmacSHA256(serviceName, kRegion.binary());
		let kSigning = hmacSHA256("aws4_request", kService.binary());

		assert.equal(expectedKDate, kDate.hex());
		assert.equal(expectedKRegion, kRegion.hex());
		assert.equal(expectedKService, kService.hex());
		assert.equal(expectedKSigning, kSigning.hex());
		`)
	assert.NoError(t, err)
}
