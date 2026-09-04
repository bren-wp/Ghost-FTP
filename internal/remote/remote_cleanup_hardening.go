package remote

import (
	"errors"
	"fmt"
	"strings"
)

// remoteResidualArtifactError marks a remote operation whose cleanup could not
// be confirmed. Automatic retry must not repeat an operation while a previous
// .GhostFTP-part-* or rollback object may still exist on the server.
type remoteResidualArtifactError struct {
	object       string
	operationErr error
	cleanupErr   error
	committed    bool
}

func (e *remoteResidualArtifactError) Error() string {
	if e == nil {
		return "remote stanje nije moguće sigurno potvrditi"
	}
	if e.committed {
		return fmt.Sprintf("nova datoteka je aktivirana, ali privremeni remote artefakt %q nije moguće sigurno ukloniti: %v", e.object, e.cleanupErr)
	}
	return fmt.Sprintf("remote radnja nije dovršena, a privremeni artefakt %q nije moguće sigurno ukloniti: %v", e.object, e.cleanupErr)
}

func (e *remoteResidualArtifactError) Unwrap() []error {
	if e == nil {
		return nil
	}
	var errs []error
	if e.operationErr != nil {
		errs = append(errs, e.operationErr)
	}
	if e.cleanupErr != nil {
		errs = append(errs, e.cleanupErr)
	}
	return errs
}

func isRemoteResidualArtifactError(err error) bool {
	var residual *remoteResidualArtifactError
	return errors.As(err, &residual)
}

// HasUncertainRemoteState lets the transfer lifecycle distinguish an ordinary
// cancel/skip from a cancel/skip whose remote staging cleanup could not be
// confirmed. The latter must surface as failed instead of looking harmless.
func HasUncertainRemoteState(err error) bool {
	return isRemoteResidualArtifactError(err)
}

// A failed upload can legitimately fail before the remote staging object is
// created. In that case a cleanup delete returning a precise not-found result
// confirms there is no residual object and must not hide the original error.
func remoteCleanupConfirmsMissing(err error) bool {
	if err == nil {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{"no such file", "does not exist", "not found"} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func cleanupRemoteArtifact(dir, name string, delete remoteDeleteFunc) error {
	if delete == nil {
		return errors.New("remote cleanup nije dostupan")
	}
	cleanupCtx, cancel := cleanupContext()
	defer cancel()
	err := delete(cleanupCtx, dir, name, false)
	if remoteCleanupConfirmsMissing(err) {
		return nil
	}
	return err
}

func cleanupFailure(operationErr error, dir, name string, delete remoteDeleteFunc) error {
	cleanupErr := cleanupRemoteArtifact(dir, name, delete)
	if cleanupErr == nil {
		return operationErr
	}
	return &remoteResidualArtifactError{
		object:       remoteJoin(dir, name),
		operationErr: operationErr,
		cleanupErr:   cleanupErr,
	}
}

func committedCleanupFailure(operationErr error, dir, name string, delete remoteDeleteFunc) error {
	cleanupErr := cleanupRemoteArtifact(dir, name, delete)
	if cleanupErr == nil {
		return operationErr
	}
	return &remoteResidualArtifactError{
		object:       remoteJoin(dir, name),
		operationErr: operationErr,
		cleanupErr:   cleanupErr,
		committed:    true,
	}
}
