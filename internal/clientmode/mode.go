package clientmode

import "strings"

type Mode string

const (
	Suite Mode = "suite"
	FTP   Mode = "ftp"
	SFTP  Mode = "sftp"
	SSH   Mode = "ssh"
	S3    Mode = "s3"
)

func Parse(value string) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(value))) {
	case FTP:
		return FTP
	case SFTP:
		return SFTP
	case SSH:
		return SSH
	case S3:
		return S3
	default:
		return Suite
	}
}

func (m Mode) Slug() string {
	switch m {
	case FTP:
		return "ftp-client"
	case SFTP:
		return "sftp-client"
	case SSH:
		return "ssh-client"
	case S3:
		return "s3-client"
	default:
		return "suite"
	}
}

func (m Mode) ProductName() string {
	switch m {
	case FTP:
		return "ByFTP FTP Client"
	case SFTP:
		return "ByFTP SFTP Client"
	case SSH:
		return "ByFTP SSH Client"
	case S3:
		return "ByFTP S3 Client"
	default:
		return "ByFTP"
	}
}

func (m Mode) ProductDescription() string {
	switch m {
	case FTP:
		return "FTP i FTPS prijenos datoteka"
	case SFTP:
		return "SFTP prijenos datoteka preko SSH-a"
	case SSH:
		return "SSH terminal i udaljene naredbe"
	case S3:
		return "S3 objektna pohrana"
	default:
		return "FTP, FTPS i SFTP prijenos datoteka"
	}
}

func (m Mode) IsFileTransfer() bool {
	return m == Suite || m == FTP || m == SFTP
}

func (m Mode) AllowsProtocol(protocol string) bool {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	switch m {
	case FTP:
		return protocol == "ftp" || protocol == "ftps" || protocol == "ftpsi"
	case SFTP:
		return protocol == "sftp"
	case Suite:
		return protocol == "ftp" || protocol == "ftps" || protocol == "ftpsi" || protocol == "sftp"
	default:
		return false
	}
}

func (m Mode) InstanceKey() string {
	return "Brendigo.ByFTP." + m.Slug()
}
