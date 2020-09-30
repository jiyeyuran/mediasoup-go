package mediasoup

/**
 * SRTP parameters.
 */
type SrtpParameters struct {
	/**
	 * Encryption and authentication transforms to be used.
	 */
	CryptoSuite SrtpCryptoSuite `json:"cryptoSuite"`

	/**
	 * SRTP keying material (master key and salt) in Base64.
	 */
	KeyBase64 string `json:"keyBase64"`
}

/**
 * SRTP crypto suite.
 */
type SrtpCryptoSuite string

const (
	AES_CM_128_HMAC_SHA1_80 SrtpCryptoSuite = "AES_CM_128_HMAC_SHA1_80"
	AES_CM_128_HMAC_SHA1_32                 = "AES_CM_128_HMAC_SHA1_32"
)
