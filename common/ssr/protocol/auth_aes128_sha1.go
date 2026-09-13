package protocol

import "github.com/sagernet/sing-box/common/ssr/tools"

func init() {
	register("auth_aes128_sha1", newAuthAES128SHA1, 9)
}

func newAuthAES128SHA1(b *Base) Protocol {
	a := &authAES128{
		Base:               b,
		authData:           &authData{},
		authAES128Function: &authAES128Function{salt: "auth_aes128_sha1", hmac: tools.HmacSHA1, hashDigest: tools.SHA1Sum},
		userData:           &userData{},
	}
	a.initUserData()
	return a
}
