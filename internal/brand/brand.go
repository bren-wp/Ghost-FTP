package brand

const (
	ProductName = "Ghost FTP"
	ProductFull = "Ghost FTP file transfer client"
	Company     = ProductName

	// Generic runtime metadata is product-only. Publisher/author identity is a
	// deliberate About-card detail and must not leak into unrelated UI, package
	// metadata or support/documentation surfaces.
	Website = "ghostftp.com"
	Support = Website

	// Explicit user actions may open only the official Ghost FTP website.
	// Update simulation is local-only and never performs a background network
	// request or sends server/account/file metadata.
	WebsiteURL = "https://ghostftp.com/"
	UpdateURL  = "https://ghostftp.com/#download"
	PremiumURL = "https://ghostftp.com/premium/"
)
