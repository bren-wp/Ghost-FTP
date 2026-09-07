package desktop

import "github.com/bren-wp/Ghost-FTP/internal/model"

// duplicateProfileDraft copies only non-secret profile configuration into a
// new profile draft. Stored credentials and the SFTP host-key pin deliberately
// do not cross the new profile boundary: credentials must be entered again and
// SFTP trust must be established for the duplicate before it can be persisted.
func duplicateProfileDraft(profile model.PublicProfile) model.ProfileInput {
	return model.ProfileInput{
		Name:           profile.Name + " 2",
		Protocol:       profile.Protocol,
		Host:           profile.Host,
		Port:           profile.Port,
		Username:       profile.Username,
		PrivateKeyPath: profile.PrivateKeyPath,
		RemotePath:     profile.RemotePath,
		LocalPath:      profile.LocalPath,
	}
}
