package desktop

// defaultConnectionProtocol is the secure fresh/quick-connect protocol on all
// supported desktop platforms. Plain FTP stays available only as an explicit
// compatibility choice; missing or invalid UI protocol state must not silently
// downgrade a new connection to cleartext FTP.
const defaultConnectionProtocol = "ftps"
