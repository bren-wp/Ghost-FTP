package brand

const (
	ProductName = "Ghost FTP"
	ProductFull = "Ghost FTP file transfer client"
	Publisher   = "BRENDIGO LTD"
	Company     = Publisher

	// Runtime metadata remains schemeless so displaying official destinations
	// never introduces an automatic network request. Product and publisher
	// identities are intentionally separate: Ghost FTP owns the product site,
	// while BRENDIGO LTD owns the author/publisher site and support contact.
	Website       = "ghostftp.com"
	AuthorWebsite = "brendigo.com"
	Support       = "brendigo.com/kontakt"
)
