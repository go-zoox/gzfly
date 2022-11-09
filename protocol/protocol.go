package protocol

// Reference:
//   SOCKS5: https://www.quarkay.com/code/383/socks5-protocol-rfc-chinese-traslation
//   SHADOWSOCKS5: https://www.ichenxiaoyu.com/ss/
//   SOCKS6: https://datatracker.ietf.org/doc/html/draft-olteanu-intarea-socks-6
//   VMESS: https://github.com/v2ray/manual/blob/master/eng_en/protocols/vmess.md
//   mKCP: https://github.com/v2ray/manual/blob/master/eng_en/protocols/mkcp.md
//   MUXCOOL: https://github.com/v2ray/manual/blob/master/eng_en/protocols/muxcool.md

// USER
//  USER_ID
//  PAIR_KEY
//	USER_NAME
//  USER_PASSWORD

// PACKET Protocol:
//  VER | CMD | CRYPTO | COMPRESS | DATA
//   1  |  1  |  1     |   1      | -

// DATA Protocol:
//
// AUTHENTICATE DATA:
// request:  USER_ID | TIMESTAMP | NONCE | SIGNATURE
//             10    |    13     |   6   |  64 HMAC_SHA256
// response: STATUS | MESSAGE
//            1     |  -

// Handshake DATA:
// request:  CONNECTION_ID | TARGET_USER_ID | TARGET_USER_PAIR_KEY | ATYP | DST.ADDR 			| DST.PORT
//					       21      |       10       |					10           |   1  |   4 or 16      |    2
// response: STATUS | MESSAGE
//            1     |  -

// TRANSMISIION DATA:
// request:  CONNECTION_ID | DATA
//					       21      |  -
