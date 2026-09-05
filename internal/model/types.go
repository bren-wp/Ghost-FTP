package model

import "time"

const (
	ConflictPolicySkip          = "skip"
	ConflictPolicyReplace       = "replace"
	ConflictPolicyReplaceBackup = "replace_backup"
)

type Item struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	IsDirectory bool      `json:"isDirectory"`
	IsSymlink   bool      `json:"isSymlink"`
	Modified    time.Time `json:"modified,omitempty"`
}

type Profile struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Protocol       string `json:"protocol"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	PasswordBlob   string `json:"passwordBlob,omitempty"`
	PrivateKeyPath string `json:"privateKeyPath,omitempty"`
	PassphraseBlob string `json:"passphraseBlob,omitempty"`
	Fingerprint    string `json:"fingerprint,omitempty"`
	RemotePath     string `json:"remotePath,omitempty"`
	LocalPath      string `json:"localPath,omitempty"`
}

type PublicProfile struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Protocol       string `json:"protocol"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	HasPassword    bool   `json:"hasPassword"`
	PrivateKeyPath string `json:"privateKeyPath,omitempty"`
	HasPassphrase  bool   `json:"hasPassphrase"`
	Fingerprint    string `json:"fingerprint,omitempty"`
	RemotePath     string `json:"remotePath,omitempty"`
	LocalPath      string `json:"localPath,omitempty"`
}

type ProfileInput struct {
	ID              string `json:"id,omitempty"`
	Name            string `json:"name"`
	Protocol        string `json:"protocol"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password,omitempty"`
	ClearPassword   bool   `json:"clearPassword,omitempty"`
	PrivateKeyPath  string `json:"privateKeyPath,omitempty"`
	Passphrase      string `json:"passphrase,omitempty"`
	ClearPassphrase bool   `json:"clearPassphrase,omitempty"`
	Fingerprint     string `json:"fingerprint,omitempty"`
	RemotePath      string `json:"remotePath,omitempty"`
	LocalPath       string `json:"localPath,omitempty"`
}

type Settings struct {
	Language                 string `json:"language,omitempty"`
	Parallelism              int    `json:"parallelism"`
	ConflictPolicy           string `json:"conflictPolicy,omitempty"`
	BackupBeforeOverwrite    bool   `json:"backupBeforeOverwrite"`
	ConfirmDelete            bool   `json:"confirmDelete"`
	AutoRetryCount           int    `json:"autoRetryCount,omitempty"`
	RetryDelaySeconds        int    `json:"retryDelaySeconds,omitempty"`
	SkipExisting             bool   `json:"skipExisting,omitempty"`
	ConnectionTimeoutSeconds int    `json:"connectionTimeoutSeconds,omitempty"`
}

type ConnectionConfig struct {
	Protocol       string `json:"protocol"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password,omitempty"`
	PrivateKeyPath string `json:"privateKeyPath,omitempty"`
	Passphrase     string `json:"passphrase,omitempty"`
	Fingerprint    string `json:"fingerprint,omitempty"`
}

type TransferJob struct {
	ID         string  `json:"id"`
	Direction  string  `json:"direction"`
	LocalPath  string  `json:"localPath"`
	RemotePath string  `json:"remotePath"`
	LocalRoot  string  `json:"-"`
	Status     string  `json:"status"`
	Progress   float64 `json:"progress"`
	Attempts   int     `json:"attempts,omitempty"`
	Error      string  `json:"error,omitempty"`
	CreatedAt  string  `json:"createdAt"`
}
