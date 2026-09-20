package dns

import ("strings"
		"errors")

func ParseReceiverDomain(receiverEmailAddr string) (string, error) {

	parts := strings.Split(receiverEmailAddr, "@")
	
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {

		return "", errors.New("invalid email format")

	}

	return parts[1], nil

}
