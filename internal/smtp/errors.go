package smtp

import ("errors"
		"net/textproto")

func IsPermanentFailure(err error) bool {
	var smtpErr *textproto.Error
	if errors.As(err, &smtpErr) {
		return smtpErr.Code >= 500 && smtpErr.Code < 600
	}
	
	return false
}
