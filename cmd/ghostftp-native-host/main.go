package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

const (
	maxInboundMessage  = 1024 * 1024
	maxOutboundMessage = 900 * 1024
	operationTimeout   = 45 * time.Second
)

var version = "0.0.6"

type request struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Params json.RawMessage `json:"params,omitempty"`
}

type bridgeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type response struct {
	ID     string       `json:"id,omitempty"`
	OK     bool         `json:"ok"`
	Result any          `json:"result,omitempty"`
	Error  *bridgeError `json:"error,omitempty"`
}

type host struct {
	engine    *api.Engine
	localRoot string
}

func hasControlCharacters(value string) bool {
	return strings.ContainsAny(value, "\x00\r\n")
}

func decodeStrict[T any](raw json.RawMessage) (T, error) {
	var value T
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&value); err != nil {
		return value, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return value, errors.New("multiple JSON values are not allowed")
		}
		return value, err
	}
	return value, nil
}

func requireEmptyParams(raw json.RawMessage) error {
	_, err := decodeStrict[struct{}](raw)
	return err
}

func decodeRequest(payload []byte) (request, error) {
	var req request
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return req, errors.New("multiple JSON values are not allowed")
		}
		return req, err
	}
	if hasControlCharacters(req.ID) || hasControlCharacters(req.Type) {
		return req, errors.New("request id and type contain control characters")
	}
	req.ID = strings.TrimSpace(req.ID)
	req.Type = strings.TrimSpace(req.Type)
	if req.ID == "" || len(req.ID) > 128 || req.Type == "" || len(req.Type) > 96 {
		return req, errors.New("request id and type are required")
	}
	return req, nil
}

func readFrame(r io.Reader) ([]byte, error) {
	var size uint32
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return nil, err
	}
	if size == 0 || size > maxInboundMessage {
		return nil, errors.New("native message size is invalid")
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeFrame(w io.Writer, payload []byte) error {
	if len(payload) == 0 || len(payload) > maxOutboundMessage {
		return errors.New("native response size is invalid")
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(payload))); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func writeResponse(w io.Writer, resp response) error {
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	if len(payload) > maxOutboundMessage {
		payload, err = json.Marshal(response{
			ID: resp.ID,
			OK: false,
			Error: &bridgeError{
				Code:    "response_too_large",
				Message: "This folder contains too much data to display safely in the browser. Open a smaller folder.",
			},
		})
		if err != nil {
			return err
		}
	}
	return writeFrame(w, payload)
}

func safeFailure(id, code string, err error, fallback string) response {
	return response{
		ID: id,
		OK: false,
		Error: &bridgeError{
			Code:    code,
			Message: usererror.Message(err, fallback),
		},
	}
}

func invalidRequest(id, message string) response {
	return response{
		ID: id,
		OK: false,
		Error: &bridgeError{
			Code:    "invalid_request",
			Message: message,
		},
	}
}

func pathWithin(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func canonicalDirectory(path string) (string, error) {
	if hasControlCharacters(path) {
		return "", errors.New("invalid local folder")
	}
	path = strings.TrimSpace(path)
	if path == "" || len(path) > 32767 {
		return "", errors.New("invalid local folder")
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("selected local path is not a folder")
	}
	return filepath.Clean(resolved), nil
}

func (h *host) authorizeRoot(path string) (string, error) {
	root, err := canonicalDirectory(path)
	if err != nil {
		return "", err
	}
	h.localRoot = root
	return root, nil
}

func (h *host) existingLocalPath(path string) (string, error) {
	if h.localRoot == "" {
		return "", errors.New("choose a local folder first")
	}
	if hasControlCharacters(path) {
		return "", errors.New("invalid local path")
	}
	path = strings.TrimSpace(path)
	if path == "" || len(path) > 32767 {
		return "", errors.New("invalid local path")
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	resolved = filepath.Clean(resolved)
	if !pathWithin(h.localRoot, resolved) {
		return "", errors.New("local path is outside the selected folder")
	}
	return resolved, nil
}

func safeLeaf(name string) (string, error) {
	if hasControlCharacters(name) {
		return "", errors.New("invalid file or folder name")
	}
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || len(name) > 255 || strings.ContainsAny(name, "/\\") {
		return "", errors.New("invalid file or folder name")
	}
	return name, nil
}

func (h *host) handle(parent context.Context, req request) response {
	ctx, cancel := context.WithTimeout(parent, operationTimeout)
	defer cancel()

	switch req.Type {
	case "hello":
		if err := requireEmptyParams(req.Params); err != nil {
			return invalidRequest(req.ID, "The bridge request is invalid.")
		}
		return response{
			ID: req.ID,
			OK: true,
			Result: map[string]any{
				"brand":           "Ghost FTP",
				"version":         version,
				"protocolVersion": 1,
				"capabilities": []string{
					"profiles",
					"connect",
					"disconnect",
					"localNavigation",
					"remoteNavigation",
					"upload",
					"download",
					"rename",
					"delete",
					"createDirectory",
					"transferProgress",
					"cancel",
					"retry",
				},
			},
		}

	case "profiles.list":
		if err := requireEmptyParams(req.Params); err != nil {
			return invalidRequest(req.ID, "The profile request is invalid.")
		}
		profiles, err := h.engine.Profiles()
		if err != nil {
			return safeFailure(req.ID, "profiles_failed", err, "Unable to load saved connections.")
		}
		return response{ID: req.ID, OK: true, Result: profiles}

	case "profiles.save":
		params, err := decodeStrict[struct {
			Profile model.ProfileInput `json:"profile"`
		}](req.Params)
		if err != nil {
			return invalidRequest(req.ID, "The saved connection is invalid.")
		}
		saved, err := h.engine.SaveProfile(params.Profile)
		if err != nil {
			return safeFailure(req.ID, "profile_save_failed", err, "Unable to save this connection.")
		}
		return response{ID: req.ID, OK: true, Result: saved}

	case "profiles.delete":
		params, err := decodeStrict[struct {
			ID string `json:"id"`
		}](req.Params)
		if err != nil || hasControlCharacters(params.ID) || strings.TrimSpace(params.ID) == "" {
			return invalidRequest(req.ID, "Select a saved connection to remove.")
		}
		if err := h.engine.RemoveProfile(strings.TrimSpace(params.ID)); err != nil {
			return safeFailure(req.ID, "profile_delete_failed", err, "Unable to remove this saved connection.")
		}
		return response{ID: req.ID, OK: true, Result: map[string]bool{"removed": true}}

	case "connect":
		params, err := decodeStrict[struct {
			ProfileID           string                 `json:"profileId,omitempty"`
			Config              model.ConnectionConfig `json:"config"`
			TrustFingerprint    string                 `json:"trustFingerprint,omitempty"`
			RememberFingerprint bool                   `json:"rememberFingerprint,omitempty"`
		}](req.Params)
		if err != nil || hasControlCharacters(params.ProfileID) || hasControlCharacters(params.TrustFingerprint) {
			return invalidRequest(req.ID, "The connection settings are invalid.")
		}
		result, err := h.engine.Connect(
			ctx,
			strings.TrimSpace(params.ProfileID),
			params.Config,
			strings.TrimSpace(params.TrustFingerprint),
			params.RememberFingerprint,
		)
		if err != nil {
			return safeFailure(req.ID, "connect_failed", err, "Unable to connect. Check the server address, port, credentials and network connection.")
		}
		return response{ID: req.ID, OK: true, Result: result}

	case "trust.cancel":
		if err := requireEmptyParams(req.Params); err != nil {
			return invalidRequest(req.ID, "The trust request is invalid.")
		}
		h.engine.CancelPendingTrust()
		return response{ID: req.ID, OK: true, Result: map[string]bool{"cancelled": true}}

	case "disconnect":
		if err := requireEmptyParams(req.Params); err != nil {
			return invalidRequest(req.ID, "The disconnect request is invalid.")
		}
		if err := h.engine.Disconnect(ctx); err != nil {
			return safeFailure(req.ID, "disconnect_failed", err, "Unable to finish disconnecting safely.")
		}
		return response{ID: req.ID, OK: true, Result: map[string]bool{"connected": false}}

	case "connection.active":
		if err := requireEmptyParams(req.Params); err != nil {
			return invalidRequest(req.ID, "The connection status request is invalid.")
		}
		cfg, connected := h.engine.ActiveConnection()
		return response{ID: req.ID, OK: true, Result: map[string]any{"connected": connected, "connection": cfg}}

	case "local.chooseRoot":
		if err := requireEmptyParams(req.Params); err != nil {
			return invalidRequest(req.ID, "The local folder request is invalid.")
		}
		selected, err := h.engine.ChooseDirectory()
		if err != nil {
			return safeFailure(req.ID, "local_picker_failed", err, "Unable to open the local folder picker.")
		}
		if strings.TrimSpace(selected) == "" {
			return invalidRequest(req.ID, "No local folder was selected.")
		}
		root, err := h.authorizeRoot(selected)
		if err != nil {
			return safeFailure(req.ID, "local_root_failed", err, "The selected local folder cannot be used safely.")
		}
		path, items, err := h.engine.LocalList(ctx, root)
		if err != nil {
			return safeFailure(req.ID, "local_list_failed", err, "Unable to read the selected local folder.")
		}
		return response{
			ID: req.ID,
			OK: true,
			Result: map[string]any{
				"root":  root,
				"path":  path,
				"items": items,
			},
		}

	case "local.list":
		params, err := decodeStrict[struct {
			Path string `json:"path"`
		}](req.Params)
		if err != nil {
			return invalidRequest(req.ID, "The local folder request is invalid.")
		}
		path, err := h.existingLocalPath(params.Path)
		if err != nil {
			return safeFailure(req.ID, "local_path_denied", err, "This local path is outside the folder you selected.")
		}
		listedPath, items, err := h.engine.LocalList(ctx, path)
		if err != nil {
			return safeFailure(req.ID, "local_list_failed", err, "Unable to read this local folder.")
		}
		return response{
			ID: req.ID,
			OK: true,
			Result: map[string]any{
				"root":  h.localRoot,
				"path":  listedPath,
				"items": items,
			},
		}

	case "local.mkdir", "local.rename", "local.delete":
		params, err := decodeStrict[struct {
			Base    string `json:"base"`
			Name    string `json:"name,omitempty"`
			NewName string `json:"newName,omitempty"`
		}](req.Params)
		if err != nil {
			return invalidRequest(req.ID, "The local file operation is invalid.")
		}
		base, err := h.existingLocalPath(params.Base)
		if err != nil {
			return safeFailure(req.ID, "local_path_denied", err, "This local path is outside the folder you selected.")
		}
		name, err := safeLeaf(params.Name)
		if err != nil {
			return invalidRequest(req.ID, "Enter a valid file or folder name.")
		}
		switch req.Type {
		case "local.mkdir":
			err = h.engine.LocalMkdir(base, name)
		case "local.rename":
			newName, nameErr := safeLeaf(params.NewName)
			if nameErr != nil {
				return invalidRequest(req.ID, "Enter a valid new name.")
			}
			err = h.engine.LocalRename(base, name, newName)
		case "local.delete":
			err = h.engine.LocalDelete(base, name)
		}
		if err != nil {
			return safeFailure(req.ID, "local_operation_failed", err, "Unable to complete the local file operation.")
		}
		return response{ID: req.ID, OK: true, Result: map[string]bool{"completed": true}}

	case "remote.list":
		params, err := decodeStrict[struct {
			Path string `json:"path"`
		}](req.Params)
		if err != nil || hasControlCharacters(params.Path) {
			return invalidRequest(req.ID, "The remote folder request is invalid.")
		}
		path := strings.TrimSpace(params.Path)
		items, err := h.engine.RemoteList(ctx, path)
		if err != nil {
			return safeFailure(req.ID, "remote_list_failed", err, "Unable to read this server folder.")
		}
		return response{ID: req.ID, OK: true, Result: map[string]any{"path": path, "items": items}}

	case "remote.mkdir", "remote.rename", "remote.delete":
		params, err := decodeStrict[struct {
			Base        string `json:"base"`
			Name        string `json:"name,omitempty"`
			NewName     string `json:"newName,omitempty"`
			IsDirectory bool   `json:"isDirectory,omitempty"`
		}](req.Params)
		if err != nil || hasControlCharacters(params.Base) {
			return invalidRequest(req.ID, "The remote file operation is invalid.")
		}
		name, err := safeLeaf(params.Name)
		if err != nil {
			return invalidRequest(req.ID, "Enter a valid file or folder name.")
		}
		base := strings.TrimSpace(params.Base)
		switch req.Type {
		case "remote.mkdir":
			err = h.engine.RemoteMkdir(ctx, base, name)
		case "remote.rename":
			newName, nameErr := safeLeaf(params.NewName)
			if nameErr != nil {
				return invalidRequest(req.ID, "Enter a valid new name.")
			}
			err = h.engine.RemoteRename(ctx, base, name, newName)
		case "remote.delete":
			err = h.engine.RemoteDelete(ctx, base, name, params.IsDirectory)
		}
		if err != nil {
			return safeFailure(req.ID, "remote_operation_failed", err, "Unable to complete the server file operation.")
		}
		return response{ID: req.ID, OK: true, Result: map[string]bool{"completed": true}}

	case "transfer.upload":
		params, err := decodeStrict[struct {
			LocalPath  string `json:"localPath"`
			RemotePath string `json:"remotePath"`
		}](req.Params)
		if err != nil || hasControlCharacters(params.RemotePath) {
			return invalidRequest(req.ID, "The upload request is invalid.")
		}
		localPath, err := h.existingLocalPath(params.LocalPath)
		if err != nil {
			return safeFailure(req.ID, "local_path_denied", err, "Select a file inside the local folder you opened.")
		}
		info, err := os.Stat(localPath)
		if err != nil || !info.Mode().IsRegular() {
			return invalidRequest(req.ID, "Select a regular local file to upload.")
		}
		job, err := h.engine.AddTransfer("upload", localPath, strings.TrimSpace(params.RemotePath), h.localRoot)
		if err != nil {
			return safeFailure(req.ID, "upload_failed", err, "Unable to add this upload to the transfer queue.")
		}
		return response{ID: req.ID, OK: true, Result: job}

	case "transfer.download":
		params, err := decodeStrict[struct {
			LocalDirectory string `json:"localDirectory"`
			RemotePath     string `json:"remotePath"`
			FileName       string `json:"fileName"`
		}](req.Params)
		if err != nil || hasControlCharacters(params.RemotePath) {
			return invalidRequest(req.ID, "The download request is invalid.")
		}
		localDir, err := h.existingLocalPath(params.LocalDirectory)
		if err != nil {
			return safeFailure(req.ID, "local_path_denied", err, "Choose a destination inside the local folder you opened.")
		}
		info, err := os.Stat(localDir)
		if err != nil || !info.IsDir() {
			return invalidRequest(req.ID, "Choose a valid local destination folder.")
		}
		fileName, err := safeLeaf(params.FileName)
		if err != nil {
			return invalidRequest(req.ID, "The downloaded file name is invalid.")
		}
		localPath := filepath.Join(localDir, fileName)
		if !pathWithin(h.localRoot, localPath) {
			return invalidRequest(req.ID, "The download destination is outside the selected local folder.")
		}
		job, err := h.engine.AddTransfer("download", localPath, strings.TrimSpace(params.RemotePath), h.localRoot)
		if err != nil {
			return safeFailure(req.ID, "download_failed", err, "Unable to add this download to the transfer queue.")
		}
		return response{ID: req.ID, OK: true, Result: job}

	case "transfer.events":
		params, err := decodeStrict[struct {
			Since int64 `json:"since"`
		}](req.Params)
		if err != nil || params.Since < 0 {
			return invalidRequest(req.ID, "The transfer status request is invalid.")
		}
		events, next := h.engine.TransferEvents(params.Since)
		return response{ID: req.ID, OK: true, Result: map[string]any{"events": events, "next": next}}

	case "transfer.cancel", "transfer.retry":
		params, err := decodeStrict[struct {
			ID string `json:"id"`
		}](req.Params)
		if err != nil || hasControlCharacters(params.ID) || strings.TrimSpace(params.ID) == "" {
			return invalidRequest(req.ID, "Select a transfer first.")
		}
		transferID := strings.TrimSpace(params.ID)
		if req.Type == "transfer.cancel" {
			err = h.engine.CancelTransfer(transferID)
		} else {
			err = h.engine.RetryTransfer(transferID)
		}
		if err != nil {
			return safeFailure(req.ID, "transfer_operation_failed", err, "Unable to update this transfer.")
		}
		return response{ID: req.ID, OK: true, Result: map[string]bool{"completed": true}}

	case "transfer.pause":
		if err := requireEmptyParams(req.Params); err != nil {
			return invalidRequest(req.ID, "The transfer pause request is invalid.")
		}
		h.engine.PauseTransfers()
		return response{ID: req.ID, OK: true, Result: map[string]bool{"paused": true}}

	case "transfer.resume":
		if err := requireEmptyParams(req.Params); err != nil {
			return invalidRequest(req.ID, "The transfer resume request is invalid.")
		}
		h.engine.ResumeTransfers()
		return response{ID: req.ID, OK: true, Result: map[string]bool{"paused": false}}

	case "transfer.clearFinished":
		if err := requireEmptyParams(req.Params); err != nil {
			return invalidRequest(req.ID, "The transfer cleanup request is invalid.")
		}
		h.engine.ClearFinishedTransfers()
		return response{ID: req.ID, OK: true, Result: map[string]bool{"completed": true}}

	default:
		return invalidRequest(req.ID, "This Ghost FTP bridge operation is not supported.")
	}
}

func main() {
	dataDir, err := api.DataDir()
	if err != nil {
		return
	}
	executable, err := os.Executable()
	if err != nil {
		return
	}
	engine, err := api.New(dataDir, executable)
	if err != nil {
		return
	}
	defer engine.Close()

	h := &host{engine: engine}
	for {
		payload, err := readFrame(os.Stdin)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return
		}
		if err != nil {
			return
		}
		req, err := decodeRequest(payload)
		if err != nil {
			if writeErr := writeResponse(os.Stdout, invalidRequest("", "The Ghost FTP bridge request is invalid.")); writeErr != nil {
				return
			}
			continue
		}
		if err := writeResponse(os.Stdout, h.handle(context.Background(), req)); err != nil {
			return
		}
	}
}
