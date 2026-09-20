package smtp

import ("errors"
		"net/textproto")

// type Error struct {

// 	Code int
// 	Msg string	

// }

func ClassifyError(err error) (code int, isSMTPError bool) {
	var smtpErr *textproto.Error
	if errors.As(err, &smtpErr) {
		return smtpErr.Code, true
	}
	return 0, false
}

func IsPermanentFailure(err error) bool {
	var smtpErr *textproto.Error
	if errors.As(err, &smtpErr) {
		return smtpErr.Code >= 500 && smtpErr.Code < 600
	}
	
	return false
}
