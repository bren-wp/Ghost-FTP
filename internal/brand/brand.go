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

	// Trusted public destinations used by explicit user actions only.
	// The app never auto-opens these URLs and never sends credentials or
	// connection metadata with update/premium requests.
	WebsiteURL        = "https://ghostftp.com/"
	PremiumURL        = "https://ghostftp.com/premium/"
	ReleaseURL        = "https://github.com/bren-wp/Ghost-FTP/releases/latest"
	ReleaseAPIURL     = "https://api.github.com/repos/bren-wp/Ghost-FTP/releases/latest"
)
