package xrpl

const (
	// XRPLHDPath is the hd path used to derive xrpl keys.
	XRPLHDPath = "m/44'/144'/0'/0/0"
	// ReserveToActivateAccount is reserve required for the account activation.
	ReserveToActivateAccount = float64(10)
	// ReservePerItem defines reserves of objects that count towards their owner's reserve requirement include:
	//	Checks, Deposit Preauthorizations, Escrows, NFT Offers, NFT Pages, Offers, Payment Channels, Signer Lists,
	//	Tickets, and Trust Lines.
	ReservePerItem = float64(2)
	// DefaultXRPLBaseFee is default XRPL base fee used for transactions.
	DefaultXRPLBaseFee = uint32(10)
)
