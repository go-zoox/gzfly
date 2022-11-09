package protocol

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
