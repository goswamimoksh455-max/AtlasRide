package offer

import _ "embed"

//go:embed lua/accept_offer.lua
var acceptOfferLua string

func loadAcceptLua() string {
	return acceptOfferLua
}
