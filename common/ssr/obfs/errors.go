package obfs

import "errors"

var (
	errTLS12TicketAuthIncorrectMagicNumber = errors.New("tls1.2_ticket_auth incorrect magic number")
	errTLS12TicketAuthTooShortData         = errors.New("tls1.2_ticket_auth too short data")
	errTLS12TicketAuthHMACError            = errors.New("tls1.2_ticket_auth hmac verifying failed")
)

type authData struct {
	clientID [32]byte
}
